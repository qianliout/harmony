package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	eventstore "harmony/component/event/store"
	personstore "harmony/component/person/store"
	recordservice "harmony/component/record/service"
	recordstore "harmony/component/record/store"
	settingstore "harmony/component/setting/store"
	viewstore "harmony/component/view/store"
	"harmony/infra/consts"
	"harmony/infra/middleware"
	"harmony/infra/model/event"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/model/view"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type ViewSrv struct {
	rdb       *rdb.RDB
	store     *viewstore.Store
	records   *recordstore.Store
	recordSrv *recordservice.RecordSrv
	persons   *personstore.Store
	events    *eventstore.Store
	settings  *settingstore.Store
}

func NewViewSrv(
	database *rdb.RDB,
	viewStore *viewstore.Store,
	recordStore *recordstore.Store,
	recordSrv *recordservice.RecordSrv,
	personStore *personstore.Store,
	eventStore *eventstore.Store,
	settingStore *settingstore.Store,
) *ViewSrv {
	return &ViewSrv{
		rdb:       database,
		store:     viewStore,
		records:   recordStore,
		recordSrv: recordSrv,
		persons:   personStore,
		events:    eventStore,
		settings:  settingStore,
	}
}

func (s *ViewSrv) PersonTimeline(ctx context.Context, q record.Query) (*view.Timeline, error) {
	ownerID := middleware.OwnerID(ctx)
	if q.PersonID == 0 {
		return nil, response.BadRequest("缺少亲友 id", "personId is required")
	}
	if _, err := s.persons.Get(s.rdb.Get(), ownerID, q.PersonID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("亲友不存在")
		}
		return nil, err
	}
	if q.Keyword != "" {
		q.Intent = record.ParseIntent(q.Keyword)
	}
	items, _, err := s.records.List(s.rdb.Get(), ownerID, q, nil)
	if err != nil {
		return nil, err
	}
	resps, err := s.recordSrv.Serialize(ctx, items)
	if err != nil {
		return nil, err
	}
	eventIDs := make([]uint64, 0, len(items))
	for i := range items {
		eventIDs = append(eventIDs, items[i].EventID)
	}
	events, err := s.events.GetMany(s.rdb.Get(), ownerID, dedupe(eventIDs))
	if err != nil {
		return nil, err
	}
	eventByID := make(map[uint64]event.Event, len(events))
	for i := range events {
		eventByID[events[i].ID] = events[i]
	}

	groupIndex := make(map[uint64]int)
	groups := make([]view.TimelineGroup, 0)
	for i := range resps {
		item := resps[i]
		index, ok := groupIndex[item.EventID]
		if !ok {
			eventItem := eventByID[item.EventID]
			groups = append(groups, view.TimelineGroup{
				Event: view.EventBrief{
					ID:        eventItem.ID,
					Name:      eventItem.Name,
					Type:      eventItem.Type,
					EventDate: utils.DateStringPtr(eventItem.EventDate),
					Host:      eventItem.Host,
					Location:  eventItem.Location,
				},
				Parts:   []view.Part{},
				Records: []record.Resp{},
			})
			index = len(groups) - 1
			groupIndex[item.EventID] = index
		}
		group := &groups[index]
		if item.Direction == consts.DirectionIn {
			group.InCent += item.TotalCent
		} else {
			group.OutCent += item.TotalCent
		}
		group.Records = append(group.Records, item)
	}
	for i := range groups {
		group := &groups[i]
		parts := make(map[string]int64)
		order := make([]string, 0)
		for j := range group.Records {
			label := group.Records[j].ItemKindLabel
			if _, ok := parts[label]; !ok {
				order = append(order, label)
			}
			parts[label] += group.Records[j].TotalCent
		}
		for _, label := range order {
			group.Parts = append(group.Parts, view.Part{Label: label, Cent: parts[label]})
		}
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return groupSortDate(groups[i]) > groupSortDate(groups[j])
	})
	return &view.Timeline{Groups: groups}, nil
}

func groupSortDate(group view.TimelineGroup) int64 {
	if group.Event.EventDate != "" {
		parsed, err := utils.ParseDate(group.Event.EventDate)
		if err == nil {
			return parsed.Unix()
		}
	}
	var latest time.Time
	for i := range group.Records {
		parsed, err := utils.ParseDate(group.Records[i].OccurredOn)
		if err == nil && parsed.After(latest) {
			latest = parsed
		}
	}
	if latest.IsZero() {
		return 0
	}
	return latest.Unix()
}

func (s *ViewSrv) EventSummary(ctx context.Context, eventID uint64) (*view.EventSummary, error) {
	ownerID := middleware.OwnerID(ctx)
	eventItem, err := s.events.Get(s.rdb.Get(), ownerID, eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("事项不存在")
		}
		return nil, err
	}
	items, _, err := s.records.List(s.rdb.Get(), ownerID, record.Query{EventID: eventID}, nil)
	if err != nil {
		return nil, err
	}
	resps, err := s.recordSrv.Serialize(ctx, items)
	if err != nil {
		return nil, err
	}
	personIDs := make([]uint64, 0, len(resps))
	for i := range resps {
		personIDs = append(personIDs, resps[i].PersonID)
	}
	persons, err := s.persons.GetMany(s.rdb.Get(), ownerID, dedupe(personIDs))
	if err != nil {
		return nil, err
	}
	personByID := make(map[uint64]person.Person, len(persons))
	for i := range persons {
		personByID[persons[i].ID] = persons[i]
	}

	summary := event.Summary{RecordCount: int64(len(resps))}
	personIndex := make(map[uint64]int)
	personOrder := make([]uint64, 0)
	personMap := make(map[uint64]*view.EventPerson)
	detailIndex := make(map[string]int)
	for i := range resps {
		item := resps[i]
		if item.Direction == consts.DirectionIn {
			summary.InCent += item.TotalCent
			switch item.Status {
			case consts.RecordStatusRepaid:
				summary.RepaidCount++
			case consts.RecordStatusPending:
				summary.PendingCount++
			}
		} else {
			summary.OutCent += item.TotalCent
		}
		if _, ok := personIndex[item.PersonID]; !ok {
			personIndex[item.PersonID] = len(personOrder)
			personOrder = append(personOrder, item.PersonID)
			personItem := personByID[item.PersonID]
			personMap[item.PersonID] = &view.EventPerson{
				Person:  personItem.Ref(),
				Details: []view.PersonDetailLine{},
			}
		}
		entry := personMap[item.PersonID]
		if item.Direction == consts.DirectionIn {
			entry.InCent += item.TotalCent
			if item.Status == consts.RecordStatusPending {
				entry.PendingCount++
			}
		} else {
			entry.OutCent += item.TotalCent
		}
		key := detailKey(item.PersonID, item.BookID, item.ItemKind, item.Direction)
		index, ok := detailIndex[key]
		if !ok {
			entry.Details = append(entry.Details, view.PersonDetailLine{
				BookName:      item.BookName,
				ItemKindLabel: item.ItemKindLabel,
				Direction:     item.Direction,
				RecordIDs:     []uint64{},
			})
			index = len(entry.Details) - 1
			detailIndex[key] = index
		}
		entry.Details[index].Cent += item.TotalCent
		entry.Details[index].RecordIDs = append(entry.Details[index].RecordIDs, item.ID)
	}
	summary.PersonCount = int64(len(personOrder))
	personsOut := make([]view.EventPerson, 0, len(personOrder))
	for _, id := range personOrder {
		entry := personMap[id]
		sort.SliceStable(entry.Details, func(i, j int) bool {
			if entry.Details[i].BookName != entry.Details[j].BookName {
				return entry.Details[i].BookName < entry.Details[j].BookName
			}
			return entry.Details[i].ItemKindLabel < entry.Details[j].ItemKindLabel
		})
		personsOut = append(personsOut, *entry)
	}
	sort.SliceStable(personsOut, func(i, j int) bool {
		if personsOut[i].InCent != personsOut[j].InCent {
			return personsOut[i].InCent > personsOut[j].InCent
		}
		return personsOut[i].OutCent > personsOut[j].OutCent
	})
	return &view.EventSummary{
		Event: view.EventBrief{
			ID:        eventItem.ID,
			Name:      eventItem.Name,
			Type:      eventItem.Type,
			EventDate: utils.DateStringPtr(eventItem.EventDate),
			Host:      eventItem.Host,
			Location:  eventItem.Location,
		},
		Summary: summary,
		Persons: personsOut,
	}, nil
}

func detailKey(personID, bookID uint64, itemKind, direction string) string {
	return strconv.FormatUint(personID, 10) + "|" + strconv.FormatUint(bookID, 10) + "|" + itemKind + "|" + direction
}

func (s *ViewSrv) Pending(ctx context.Context, kind string) (*view.PendingList, error) {
	ownerID := middleware.OwnerID(ctx)
	var rows []viewstore.PendingRow
	var err error
	switch kind {
	case "owed":
		rows, err = s.store.Owed(s.rdb.Get(), ownerID)
	case "owing":
		settings, settingsErr := s.settings.Get(s.rdb.Get(), ownerID)
		if settingsErr == nil && settings[consts.SettingShowOwing] == "false" {
			return &view.PendingList{Persons: []view.PendingPerson{}}, nil
		}
		rows, err = s.store.Owing(s.rdb.Get(), ownerID)
	default:
		return nil, response.BadRequest("待还类型不正确", "invalid pending kind")
	}
	if err != nil {
		return nil, err
	}
	personIDs := make([]uint64, 0, len(rows))
	for i := range rows {
		personIDs = append(personIDs, rows[i].PersonID)
	}
	persons, err := s.persons.GetMany(s.rdb.Get(), ownerID, personIDs)
	if err != nil {
		return nil, err
	}
	personByID := make(map[uint64]person.Person, len(persons))
	for i := range persons {
		personByID[persons[i].ID] = persons[i]
	}
	list := view.PendingList{Persons: make([]view.PendingPerson, 0, len(rows))}
	for i := range rows {
		personItem := personByID[rows[i].PersonID]
		ref := personItem.Ref()
		list.Persons = append(list.Persons, view.PendingPerson{
			Person:    ref,
			Count:     rows[i].Count,
			TotalCent: rows[i].TotalCent,
			Earliest:  utils.DateString(rows[i].Earliest),
		})
		list.TotalCount += rows[i].Count
		list.TotalCent += rows[i].TotalCent
	}
	list.PersonCount = int64(len(list.Persons))
	return &list, nil
}

func dedupe(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok || id == 0 {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
