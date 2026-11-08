package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"harmony/component/event/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/event"
	"harmony/infra/model/person"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type PersonStore interface {
	Get(db *gorm.DB, ownerID uint64, id uint64) (*person.Person, error)
	GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]person.Person, error)
}

type EventSrv struct {
	rdb     *rdb.RDB
	store   *store.Store
	persons PersonStore
	log     *logger.Logger
}

func NewEventSrv(database *rdb.RDB, eventStore *store.Store, personStore PersonStore) *EventSrv {
	return &EventSrv{
		rdb:     database,
		store:   eventStore,
		persons: personStore,
		log:     logger.New(logger.WithModule("event"), logger.WithSubModule("service")),
	}
}

func (s *EventSrv) List(ctx context.Context, q event.Query, f *modelfilter.Filter) ([]event.Resp, int64, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Type != "" && !consts.ValidEventType(q.Type) {
		return nil, 0, response.BadRequest("事项类型不正确", "invalid event type")
	}
	if q.Host != "" && !consts.ValidEventHost(q.Host) {
		return nil, 0, response.BadRequest("主办方取值不正确", "invalid event host")
	}
	ownerID := middleware.OwnerID(ctx)
	items, total, err := s.store.List(s.rdb.Get(), ownerID, q, f)
	if err != nil {
		return nil, 0, err
	}
	out, err := s.serializeMany(ownerID, items)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *EventSrv) Recent(ctx context.Context, limit int) ([]event.Resp, error) {
	ownerID := middleware.OwnerID(ctx)
	items, err := s.store.Recent(s.rdb.Get(), ownerID, limit)
	if err != nil {
		return nil, err
	}
	return s.serializeMany(ownerID, items)
}

func (s *EventSrv) Detail(ctx context.Context, id uint64) (*event.Detail, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("事项不存在")
		}
		return nil, err
	}
	out, err := s.serializeMany(ownerID, []event.Event{*item})
	if err != nil {
		return nil, err
	}
	return &event.Detail{Resp: out[0], Summary: event.Summary{
		InCent:       out[0].InCent,
		OutCent:      out[0].OutCent,
		PersonCount:  out[0].PersonCount,
		RepaidCount:  out[0].RepaidCount,
		PendingCount: out[0].PendingCount,
		RecordCount:  out[0].RecordCount,
	}}, nil
}

func (s *EventSrv) Create(ctx context.Context, param event.CreateParam) (*event.Detail, error) {
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil, response.BadRequest("事项名称不能为空", "event name is required")
	}
	ownerID := middleware.OwnerID(ctx)
	eventType := strings.TrimSpace(param.Type)
	if eventType == "" {
		eventType = consts.EventTypeOther
	}
	if !consts.ValidEventType(eventType) {
		return nil, response.BadRequest("事项类型不正确", "invalid event type")
	}
	host := strings.TrimSpace(param.Host)
	if host == "" {
		host = consts.EventHostOther
	}
	if !consts.ValidEventHost(host) {
		return nil, response.BadRequest("主办方取值不正确", "invalid event host")
	}
	precision := strings.TrimSpace(param.DatePrecision)
	if precision == "" {
		precision = consts.DatePrecisionDay
	}
	if !consts.ValidDatePrecision(precision) {
		return nil, response.BadRequest("日期精度不正确", "invalid date precision")
	}
	var eventDate *string
	eventDate = nil
	if strings.TrimSpace(param.EventDate) != "" {
		parsed, err := utils.ParseDate(param.EventDate)
		if err != nil {
			return nil, response.BadRequest("日期格式不正确", "invalid date")
		}
		normalized := utils.DateString(utils.NormalizeDate(parsed, precision))
		eventDate = &normalized
	}
	if param.HostPersonID > 0 {
		if _, err := s.persons.Get(s.rdb.Get(), ownerID, param.HostPersonID); err != nil {
			return nil, response.BadRequest("主办人不存在", "invalid host person")
		}
	}
	item := &event.Event{
		OwnerID:       ownerID,
		Name:          name,
		Type:          eventType,
		Location:      strings.TrimSpace(param.Location),
		Host:          host,
		DatePrecision: precision,
		Remark:        strings.TrimSpace(param.Remark),
	}
	if eventDate != nil {
		parsed, _ := utils.ParseDate(*eventDate)
		item.EventDate = &parsed
	}
	if param.HostPersonID > 0 {
		item.HostPersonID = &param.HostPersonID
	}
	if err := s.store.Create(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("id", item.ID).Msg("event.create")
	return s.Detail(ctx, item.ID)
}

func (s *EventSrv) Update(ctx context.Context, id uint64, param event.UpdateParam) (*event.Detail, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("事项不存在")
		}
		return nil, err
	}
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil, response.BadRequest("事项名称不能为空", "event name is required")
	}
	eventType := strings.TrimSpace(param.Type)
	if eventType == "" {
		eventType = item.Type
	}
	if !consts.ValidEventType(eventType) {
		return nil, response.BadRequest("事项类型不正确", "invalid event type")
	}
	host := strings.TrimSpace(param.Host)
	if host == "" {
		host = item.Host
	}
	if !consts.ValidEventHost(host) {
		return nil, response.BadRequest("主办方取值不正确", "invalid event host")
	}
	precision := strings.TrimSpace(param.DatePrecision)
	if precision == "" {
		precision = item.DatePrecision
	}
	if !consts.ValidDatePrecision(precision) {
		return nil, response.BadRequest("日期精度不正确", "invalid date precision")
	}
	item.Name = name
	item.Type = eventType
	item.Host = host
	item.DatePrecision = precision
	item.Location = strings.TrimSpace(param.Location)
	item.Remark = strings.TrimSpace(param.Remark)
	if strings.TrimSpace(param.EventDate) == "" {
		item.EventDate = nil
	} else {
		parsed, err := utils.ParseDate(param.EventDate)
		if err != nil {
			return nil, response.BadRequest("日期格式不正确", "invalid date")
		}
		normalized := utils.NormalizeDate(parsed, precision)
		item.EventDate = &normalized
	}
	if param.HostPersonID > 0 {
		if _, err := s.persons.Get(s.rdb.Get(), ownerID, param.HostPersonID); err != nil {
			return nil, response.BadRequest("主办人不存在", "invalid host person")
		}
		item.HostPersonID = &param.HostPersonID
	} else {
		item.HostPersonID = nil
	}
	if err := s.store.Update(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("event.update")
	return s.Detail(ctx, id)
}

func (s *EventSrv) Delete(ctx context.Context, id uint64) error {
	ownerID := middleware.OwnerID(ctx)
	if _, err := s.store.Get(s.rdb.Get(), ownerID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("事项不存在")
		}
		return err
	}
	count, err := s.store.CountRecords(s.rdb.Get(), ownerID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return response.HasReference("这个事项下还有记录，先处理记录再删除", count)
	}
	if err := s.store.Delete(s.rdb.Get(), ownerID, id); err != nil {
		return err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("event.delete")
	return nil
}

func (s *EventSrv) serializeMany(ownerID uint64, items []event.Event) ([]event.Resp, error) {
	eventIDs := make([]uint64, 0, len(items))
	hostIDs := make([]uint64, 0, len(items))
	for i := range items {
		eventIDs = append(eventIDs, items[i].ID)
		if items[i].HostPersonID != nil {
			hostIDs = append(hostIDs, *items[i].HostPersonID)
		}
	}
	agg, err := s.store.Aggregates(s.rdb.Get(), ownerID, eventIDs)
	if err != nil {
		return nil, err
	}
	hostNames := make(map[uint64]string, len(hostIDs))
	if len(hostIDs) > 0 {
		persons, err := s.persons.GetMany(s.rdb.Get(), ownerID, hostIDs)
		if err != nil {
			return nil, err
		}
		for i := range persons {
			hostNames[persons[i].ID] = persons[i].Name
		}
	}
	out := make([]event.Resp, 0, len(items))
	for i := range items {
		item := items[i]
		summary := agg[item.ID]
		resp := event.Resp{
			ID:            item.ID,
			Name:          item.Name,
			Type:          item.Type,
			TypeLabel:     event.LabelOfType(item.Type),
			EventDate:     utils.DateStringPtr(item.EventDate),
			DatePrecision: item.DatePrecision,
			Location:      item.Location,
			Host:          item.Host,
			HostLabel:     event.LabelOfHost(item.Host),
			Placeholder:   item.Placeholder,
			Remark:        item.Remark,
			RecordCount:   summary.RecordCount,
			InCent:        summary.InCent,
			OutCent:       summary.OutCent,
			PersonCount:   summary.PersonCount,
			RepaidCount:   summary.RepaidCount,
			PendingCount:  summary.PendingCount,
			CreatedAt:     utils.Millis(item.CreatedAt),
			UpdatedAt:     utils.Millis(item.UpdatedAt),
		}
		if item.HostPersonID != nil {
			resp.HostPersonID = *item.HostPersonID
			resp.HostPersonName = hostNames[*item.HostPersonID]
		}
		out = append(out, resp)
	}
	return out, nil
}
