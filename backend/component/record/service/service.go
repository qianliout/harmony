package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	bookstore "harmony/component/book/store"
	dictstore "harmony/component/dict/store"
	eventstore "harmony/component/event/store"
	personstore "harmony/component/person/store"
	recordstore "harmony/component/record/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/book"
	"harmony/infra/model/dict"
	"harmony/infra/model/event"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type PersonStore interface {
	Get(db *gorm.DB, ownerID uint64, id uint64) (*person.Person, error)
	GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]person.Person, error)
	FindExactByName(db *gorm.DB, ownerID uint64, name string) ([]person.Person, error)
	Create(db *gorm.DB, p *person.Person) error
}

type EventStore interface {
	Get(db *gorm.DB, ownerID uint64, id uint64) (*event.Event, error)
	GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]event.Event, error)
	FindByHostAndName(db *gorm.DB, ownerID uint64, hostPersonID uint64, name string) (*event.Event, error)
	Create(db *gorm.DB, item *event.Event) error
	Placeholder(db *gorm.DB, ownerID uint64) (*event.Event, error)
}

type BookStore interface {
	Get(db *gorm.DB, ownerID uint64, id uint64) (*book.Book, error)
	GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]book.Book, error)
}

type DictStore interface {
	GetByCode(db *gorm.DB, ownerID uint64, dictType, code string) (*dict.Item, error)
	List(db *gorm.DB, ownerID uint64, dictType string) ([]dict.Item, error)
}

type RecordSrv struct {
	rdb     *rdb.RDB
	store   *recordstore.Store
	persons PersonStore
	events  EventStore
	books   BookStore
	dicts   DictStore
	log     *logger.Logger
}

func NewRecordSrv(
	database *rdb.RDB,
	recordStore *recordstore.Store,
	personStore *personstore.Store,
	eventStore *eventstore.Store,
	bookStore *bookstore.Store,
	dictStore *dictstore.Store,
) *RecordSrv {
	return &RecordSrv{
		rdb:     database,
		store:   recordStore,
		persons: personStore,
		events:  eventStore,
		books:   bookStore,
		dicts:   dictStore,
		log:     logger.New(logger.WithModule("record"), logger.WithSubModule("service")),
	}
}

func (s *RecordSrv) List(ctx context.Context, q record.Query, f *modelfilter.Filter) ([]record.Resp, int64, error) {
	ownerID := middleware.OwnerID(ctx)
	if q.Keyword != "" {
		q.Intent = record.ParseIntent(q.Keyword)
	}
	if q.Direction != "" && !consts.ValidDirection(q.Direction) {
		return nil, 0, response.BadRequest("方向取值不正确", "invalid direction")
	}
	if q.Status != "" && !consts.ValidRecordStatus(q.Status) {
		return nil, 0, response.BadRequest("状态取值不正确", "invalid status")
	}
	items, total, err := s.store.List(s.rdb.Get(), ownerID, q, f)
	if err != nil {
		return nil, 0, err
	}
	out, err := s.serialize(s.rdb.Get(), ownerID, items)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *RecordSrv) Detail(ctx context.Context, id uint64) (*record.Detail, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("记录不存在")
		}
		return nil, err
	}
	resps, err := s.serialize(s.rdb.Get(), ownerID, []record.Record{*item})
	if err != nil {
		return nil, err
	}
	detail := &record.Detail{
		Resp:          resps[0],
		RepayRecords:  []record.Resp{},
		OriginRecords: []record.Resp{},
	}

	repayIDs := make([]uint64, 0)
	links, err := s.store.LinksByOrigin(s.rdb.Get(), ownerID, id)
	if err != nil {
		return nil, err
	}
	for i := range links {
		repayIDs = append(repayIDs, links[i].RepayRecordID)
	}
	if len(repayIDs) > 0 {
		related, err := s.store.GetMany(s.rdb.Get(), ownerID, repayIDs)
		if err != nil {
			return nil, err
		}
		detail.RepayRecords, err = s.serialize(s.rdb.Get(), ownerID, related)
		if err != nil {
			return nil, err
		}
	}

	originIDs := make([]uint64, 0)
	repayLinks, err := s.store.LinksByRepay(s.rdb.Get(), ownerID, id)
	if err != nil {
		return nil, err
	}
	for i := range repayLinks {
		originIDs = append(originIDs, repayLinks[i].OriginRecordID)
	}
	if len(originIDs) > 0 {
		related, err := s.store.GetMany(s.rdb.Get(), ownerID, originIDs)
		if err != nil {
			return nil, err
		}
		detail.OriginRecords, err = s.serialize(s.rdb.Get(), ownerID, related)
		if err != nil {
			return nil, err
		}
	}
	return detail, nil
}

// Serialize 供聚合视图与识别核对复用。
func (s *RecordSrv) Serialize(ctx context.Context, items []record.Record) ([]record.Resp, error) {
	return s.serialize(s.rdb.Get(), middleware.OwnerID(ctx), items)
}

func (s *RecordSrv) Create(ctx context.Context, param record.CreateParam) (*record.Resp, error) {
	ownerID := middleware.OwnerID(ctx)
	var created *record.Record
	err := s.rdb.Transaction(func(tx *gorm.DB) error {
		item, err := s.CreateInTx(ctx, tx, param)
		if err != nil {
			return err
		}
		created = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	resps, err := s.serialize(s.rdb.Get(), ownerID, []record.Record{*created})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

// CreateInTx 在调用方事务内新建记录，识别批量写入与导入复用同一套校验。
func (s *RecordSrv) CreateInTx(ctx context.Context, tx *gorm.DB, param record.CreateParam) (*record.Record, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.validateAndBuild(tx, ownerID, param)
	if err != nil {
		return nil, err
	}
	if err := s.store.Create(tx, item); err != nil {
		return nil, err
	}
	// 生成列由数据库计算，Create 不会回填，这里按同一口径补齐出参。
	item.TotalCent = item.AmountCent + item.GiftValueCent
	s.log.Info(ctx).Uint64("id", item.ID).Uint64("person", item.PersonID).Msg("record.create")
	return item, nil
}

func (s *RecordSrv) validateAndBuild(tx *gorm.DB, ownerID uint64, param record.CreateParam) (*record.Record, error) {
	direction := strings.TrimSpace(param.Direction)
	if direction == "" {
		direction = consts.DirectionIn
	}
	if !consts.ValidDirection(direction) {
		return nil, response.BadRequest("方向取值不正确", "invalid direction")
	}
	precision := strings.TrimSpace(param.DatePrecision)
	if precision == "" {
		precision = consts.DatePrecisionDay
	}
	if !consts.ValidDatePrecision(precision) {
		return nil, response.BadRequest("日期精度不正确", "invalid date precision")
	}
	occurredOn, err := utils.ParseDate(param.OccurredOn)
	if err != nil {
		return nil, response.BadRequest("记账日期不正确", "invalid date")
	}
	occurredOn = utils.NormalizeDate(occurredOn, precision)

	if err := validateAmount(param.AmountCent, param.GiftName, param.GiftValueCent); err != nil {
		return nil, err
	}
	person, err := s.resolvePerson(tx, ownerID, param.PersonID, param.PersonName)
	if err != nil {
		return nil, err
	}
	bookItem, err := s.getNormalBook(tx, ownerID, param.BookID)
	if err != nil {
		return nil, err
	}
	eventID := param.EventID
	// 连续录入与识别录入可能不传事项：礼簿设置默认事项时自动带出。
	if eventID == 0 && bookItem.DefaultEventID != nil {
		eventID = *bookItem.DefaultEventID
	}
	eventItem, err := s.getEvent(tx, ownerID, eventID)
	if err != nil {
		return nil, err
	}
	itemKind, err := s.resolveItemKind(tx, ownerID, bookItem, param.ItemKind, param.AmountCent, param.GiftName, param.GiftValueCent)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(param.Status)
	if status == "" {
		status = consts.RecordStatusPending
	}
	if !consts.ValidRecordStatus(status) {
		return nil, response.BadRequest("状态取值不正确", "invalid status")
	}
	source := strings.TrimSpace(param.Source)
	if source == "" {
		source = consts.RecordSourceManual
	}
	if !consts.ValidRecordSource(source) {
		return nil, response.BadRequest("记录来源不正确", "invalid source")
	}

	if !param.Confirm && source == consts.RecordSourceManual {
		duplicates, err := s.store.Duplicates(tx, ownerID, person.ID, eventItem.ID, bookItem.ID, itemKind)
		if err != nil {
			return nil, err
		}
		if len(duplicates) > 0 {
			return nil, s.duplicateError(ownerID, person, eventItem, bookItem, itemKind, duplicates)
		}
	}

	paperSeq := param.PaperSeq
	if paperSeq < 0 {
		paperSeq = 0
	}
	item := &record.Record{
		OwnerID:         ownerID,
		PersonID:        person.ID,
		EventID:         eventItem.ID,
		BookID:          bookItem.ID,
		OriginBookID:    bookItem.ID,
		ItemKind:        itemKind,
		Direction:       direction,
		AmountCent:      param.AmountCent,
		GiftName:        strings.TrimSpace(param.GiftName),
		GiftValueCent:   param.GiftValueCent,
		OccurredOn:      occurredOn,
		DatePrecision:   precision,
		Status:          status,
		Remark:          strings.TrimSpace(param.Remark),
		PaperNote:       strings.TrimSpace(param.PaperNote),
		PaperPage:       strings.TrimSpace(param.PaperPage),
		PaperSeq:        paperSeq,
		Source:          source,
		RecognizeTaskID: param.RecognizeTaskID,
	}
	return item, nil
}

func (s *RecordSrv) Update(ctx context.Context, id uint64, param record.UpdateParam) (*record.Resp, error) {
	ownerID := middleware.OwnerID(ctx)
	db := s.rdb.Get()
	item, err := s.store.Get(db, ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("记录不存在")
		}
		return nil, err
	}
	linked, err := s.hasAnyLink(db, ownerID, id)
	if err != nil {
		return nil, err
	}

	personID := item.PersonID
	if param.PersonID > 0 && param.PersonID != item.PersonID {
		if linked {
			return nil, response.StateConflict("这笔记录已有还礼关联，先解除关联再换亲友")
		}
		if _, err := s.persons.Get(db, ownerID, param.PersonID); err != nil {
			return nil, response.BadRequest("亲友不存在", "invalid person")
		}
		personID = param.PersonID
	}
	direction := item.Direction
	if param.Direction != "" && param.Direction != item.Direction {
		if linked {
			return nil, response.StateConflict("这笔记录已有还礼关联，先解除关联再改方向")
		}
		if !consts.ValidDirection(param.Direction) {
			return nil, response.BadRequest("方向取值不正确", "invalid direction")
		}
		direction = param.Direction
	}
	eventID := item.EventID
	if param.EventID > 0 && param.EventID != item.EventID {
		if _, err := s.getEvent(db, ownerID, param.EventID); err != nil {
			return nil, err
		}
		eventID = param.EventID
	}
	bookID := item.BookID
	if param.BookID > 0 && param.BookID != item.BookID {
		if _, err := s.getNormalBook(db, ownerID, param.BookID); err != nil {
			return nil, err
		}
		bookID = param.BookID
	}
	bookItem, err := s.getNormalBook(db, ownerID, bookID)
	if err != nil {
		return nil, err
	}
	if err := validateAmount(param.AmountCent, param.GiftName, param.GiftValueCent); err != nil {
		return nil, err
	}
	itemKind, err := s.resolveItemKind(db, ownerID, bookItem, param.ItemKind, param.AmountCent, param.GiftName, param.GiftValueCent)
	if err != nil {
		return nil, err
	}
	precision := strings.TrimSpace(param.DatePrecision)
	if precision == "" {
		precision = item.DatePrecision
	}
	if !consts.ValidDatePrecision(precision) {
		return nil, response.BadRequest("日期精度不正确", "invalid date precision")
	}
	occurredOn := item.OccurredOn
	if strings.TrimSpace(param.OccurredOn) != "" {
		parsed, err := utils.ParseDate(param.OccurredOn)
		if err != nil {
			return nil, response.BadRequest("记账日期不正确", "invalid date")
		}
		occurredOn = utils.NormalizeDate(parsed, precision)
	}
	status := item.Status
	if param.Status != "" {
		if !consts.ValidRecordStatus(param.Status) {
			return nil, response.BadRequest("状态取值不正确", "invalid status")
		}
		status = param.Status
	}

	item.PersonID = personID
	item.EventID = eventID
	item.BookID = bookID
	item.Direction = direction
	item.ItemKind = itemKind
	item.AmountCent = param.AmountCent
	item.GiftName = strings.TrimSpace(param.GiftName)
	item.GiftValueCent = param.GiftValueCent
	item.OccurredOn = occurredOn
	item.DatePrecision = precision
	item.Status = status
	item.Remark = strings.TrimSpace(param.Remark)
	item.PaperNote = strings.TrimSpace(param.PaperNote)
	item.PaperPage = strings.TrimSpace(param.PaperPage)
	item.PaperSeq = maxInt(param.PaperSeq, 0)
	if err := s.store.Update(db, item); err != nil {
		return nil, err
	}
	item.TotalCent = item.AmountCent + item.GiftValueCent
	s.log.Info(ctx).Uint64("id", id).Msg("record.update")
	resps, err := s.serialize(db, ownerID, []record.Record{*item})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

func (s *RecordSrv) Delete(ctx context.Context, id uint64, confirm bool) error {
	ownerID := middleware.OwnerID(ctx)
	db := s.rdb.Get()
	item, err := s.store.Get(db, ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("记录不存在")
		}
		return err
	}
	if item.Direction == consts.DirectionIn {
		count, err := s.store.CountLinksByOrigin(db, ownerID, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return response.HasReference("这笔收礼已经被还礼关联，先解除还礼", count)
		}
		return s.deleteRecord(ownerID, id)
	}

	links, err := s.store.LinksByRepay(db, ownerID, id)
	if err != nil {
		return err
	}
	if len(links) > 0 && !confirm {
		payload := response.ConfirmPayload{
			Kind:        consts.ConfirmUnlinkOnDelete,
			Title:       "删除这笔还礼",
			Message:     "这笔还礼关联了 " + strconv.Itoa(len(links)) + " 笔收礼，删除后会解除关联",
			Conflicts:   []response.ConfirmConflict{{ID: id, Summary: "送 ¥" + amountLabel(item.TotalCent) + " · " + utils.DateString(item.OccurredOn)}},
			ConfirmText: "删除并解除关联",
		}
		return response.Conflict(consts.ReasonNeedConfirm, payload.Message, payload)
	}
	return s.rdb.Transaction(func(tx *gorm.DB) error {
		for i := range links {
			originID := links[i].OriginRecordID
			if err := s.store.DeleteLink(tx, ownerID, id, originID); err != nil {
				return err
			}
			remaining, err := s.store.CountLinksByOrigin(tx, ownerID, originID)
			if err != nil {
				return err
			}
			if remaining == 0 {
				origin, err := s.store.Get(tx, ownerID, originID)
				if err != nil {
					return err
				}
				if origin.Status == consts.RecordStatusRepaid {
					if err := s.store.SetStatus(tx, ownerID, []uint64{originID}, consts.RecordStatusPending); err != nil {
						return err
					}
				}
			}
		}
		return s.store.Delete(tx, ownerID, id)
	})
}

func (s *RecordSrv) deleteRecord(ownerID uint64, id uint64) error {
	if err := s.store.Delete(s.rdb.Get(), ownerID, id); err != nil {
		return err
	}
	return nil
}

func (s *RecordSrv) Unlink(ctx context.Context, repayID uint64, originID uint64) error {
	ownerID := middleware.OwnerID(ctx)
	db := s.rdb.Get()
	if _, err := s.store.Get(db, ownerID, repayID); err != nil {
		return response.NotFound("还礼记录不存在")
	}
	links, err := s.store.LinksByRepay(db, ownerID, repayID)
	if err != nil {
		return err
	}
	found := false
	for i := range links {
		if links[i].OriginRecordID == originID {
			found = true
			break
		}
	}
	if !found {
		return response.NotFound("没有这条还礼关联")
	}
	return s.rdb.Transaction(func(tx *gorm.DB) error {
		if err := s.store.DeleteLink(tx, ownerID, repayID, originID); err != nil {
			return err
		}
		remaining, err := s.store.CountLinksByOrigin(tx, ownerID, originID)
		if err != nil {
			return err
		}
		if remaining == 0 {
			origin, err := s.store.Get(tx, ownerID, originID)
			if err != nil {
				return err
			}
			if origin.Status == consts.RecordStatusRepaid {
				return s.store.SetStatus(tx, ownerID, []uint64{originID}, consts.RecordStatusPending)
			}
		}
		return nil
	})
}

func (s *RecordSrv) Repay(ctx context.Context, param record.RepayParam) (*record.RepayResult, error) {
	ownerID := middleware.OwnerID(ctx)
	var created []record.Record
	err := s.rdb.Transaction(func(tx *gorm.DB) error {
		items, err := s.RepayInTx(ctx, tx, param)
		if err != nil {
			return err
		}
		created = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	resps, err := s.serialize(s.rdb.Get(), ownerID, created)
	if err != nil {
		return nil, err
	}
	linkCount := 0
	if len(created) > 0 {
		linkCount = len(created) * len(param.OriginIDs)
	}
	return &record.RepayResult{Records: resps, RepayCount: linkCount}, nil
}

// RepayInTx 在调用方事务内完成还礼，识别核对页的一并补录复用这个方法。
func (s *RecordSrv) RepayInTx(ctx context.Context, tx *gorm.DB, param record.RepayParam) ([]record.Record, error) {
	ownerID := middleware.OwnerID(ctx)
	originIDs := dedupeIDs(param.OriginIDs)
	if len(originIDs) == 0 {
		return nil, response.BadRequest("请选择要还的收礼记录", "origin records are required")
	}
	origins, err := s.store.LockRecords(tx, ownerID, originIDs)
	if err != nil {
		return nil, err
	}
	if len(origins) != len(originIDs) {
		return nil, response.NotFound("部分收礼记录不存在")
	}
	personID := origins[0].PersonID
	for i := range origins {
		if origins[i].Direction != consts.DirectionIn {
			return nil, response.StateConflict("只能对收礼记录做还礼")
		}
		if origins[i].Status == consts.RecordStatusNone {
			return nil, response.StateConflict("这笔记录标为无需还，不能还礼")
		}
		if origins[i].PersonID != personID {
			return nil, response.BadRequest("一次还礼只能对应同一个人", "origin records must belong to one person")
		}
	}
	if _, err := s.persons.Get(tx, ownerID, personID); err != nil {
		return nil, response.NotFound("亲友不存在")
	}
	if len(param.Items) == 0 {
		return nil, response.BadRequest("请至少填写一个还礼名目", "repay items are required")
	}

	eventItem, err := s.resolveRepayEvent(tx, ownerID, personID, param)
	if err != nil {
		return nil, err
	}
	precision := strings.TrimSpace(param.DatePrecision)
	if precision == "" {
		precision = eventItem.DatePrecision
	}
	if !consts.ValidDatePrecision(precision) {
		return nil, response.BadRequest("日期精度不正确", "invalid date precision")
	}
	occurredOn := time.Now()
	if strings.TrimSpace(param.OccurredOn) != "" {
		parsed, err := utils.ParseDate(param.OccurredOn)
		if err != nil {
			return nil, response.BadRequest("还礼日期不正确", "invalid date")
		}
		occurredOn = parsed
	} else if eventItem.EventDate != nil {
		occurredOn = *eventItem.EventDate
	}
	occurredOn = utils.NormalizeDate(occurredOn, precision)

	created := make([]record.Record, 0, len(param.Items))
	for i := range param.Items {
		item := param.Items[i]
		if err := validateAmount(item.AmountCent, item.GiftName, item.GiftValueCent); err != nil {
			return nil, err
		}
		bookID := item.BookID
		if bookID == 0 {
			bookID = origins[0].BookID
		}
		bookItem, err := s.getNormalBook(tx, ownerID, bookID)
		if err != nil {
			return nil, err
		}
		itemKind, err := s.resolveItemKind(tx, ownerID, bookItem, item.ItemKind, item.AmountCent, item.GiftName, item.GiftValueCent)
		if err != nil {
			return nil, err
		}
		rec := record.Record{
			OwnerID:       ownerID,
			PersonID:      personID,
			EventID:       eventItem.ID,
			BookID:        bookItem.ID,
			OriginBookID:  bookItem.ID,
			ItemKind:      itemKind,
			Direction:     consts.DirectionOut,
			AmountCent:    item.AmountCent,
			GiftName:      strings.TrimSpace(item.GiftName),
			GiftValueCent: item.GiftValueCent,
			OccurredOn:    occurredOn,
			DatePrecision: precision,
			Status:        consts.RecordStatusNone,
			Remark:        strings.TrimSpace(param.Remark),
			Source:        consts.RecordSourceManual,
		}
		if err := s.store.Create(tx, &rec); err != nil {
			return nil, err
		}
		rec.TotalCent = rec.AmountCent + rec.GiftValueCent
		created = append(created, rec)
	}
	for i := range created {
		for _, originID := range originIDs {
			if err := s.store.Link(tx, created[i].ID, originID, ownerID); err != nil {
				return nil, err
			}
		}
	}
	if err := s.store.SetPaperNoteIfEmpty(tx, ownerID, originIDs, strings.TrimSpace(param.PaperNote)); err != nil {
		return nil, err
	}
	markRepaid := true
	if param.MarkRepaid != nil {
		markRepaid = *param.MarkRepaid
	}
	if markRepaid {
		if err := s.store.SetStatus(tx, ownerID, originIDs, consts.RecordStatusRepaid); err != nil {
			return nil, err
		}
	}
	s.log.Info(ctx).Int("records", len(created)).Int("origins", len(originIDs)).Msg("record.repay")
	return created, nil
}

func (s *RecordSrv) resolveRepayEvent(tx *gorm.DB, ownerID uint64, personID uint64, param record.RepayParam) (*event.Event, error) {
	if param.Event != nil && param.Event.ID > 0 {
		return s.events.Get(tx, ownerID, param.Event.ID)
	}
	name := ""
	if param.Event != nil {
		name = strings.TrimSpace(param.Event.Name)
	}
	if name == "" {
		return s.events.Placeholder(tx, ownerID)
	}
	if existing, err := s.events.FindByHostAndName(tx, ownerID, personID, name); err == nil {
		return existing, nil
	}
	eventType := consts.EventTypeOther
	if param.Event != nil && param.Event.Type != "" {
		eventType = param.Event.Type
	}
	if !consts.ValidEventType(eventType) || eventType == consts.EventTypeOther {
		eventType = event.InferTypeByName(name)
	}
	precision := consts.DatePrecisionDay
	dateText := ""
	if param.Event != nil {
		if param.Event.DatePrecision != "" {
			precision = param.Event.DatePrecision
		}
		dateText = param.Event.Date
	}
	if dateText == "" {
		dateText = param.OccurredOn
	}
	item := &event.Event{
		OwnerID:       ownerID,
		Name:          name,
		Type:          eventType,
		Host:          consts.EventHostOther,
		DatePrecision: precision,
	}
	if dateText != "" {
		parsed, err := utils.ParseDate(dateText)
		if err != nil {
			return nil, response.BadRequest("还礼事项日期不正确", "invalid event date")
		}
		normalized := utils.NormalizeDate(parsed, precision)
		item.EventDate = &normalized
	}
	item.HostPersonID = &personID
	if err := s.events.Create(tx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *RecordSrv) MarkRepaid(ctx context.Context, id uint64, param record.MarkRepaidParam) error {
	ownerID := middleware.OwnerID(ctx)
	db := s.rdb.Get()
	item, err := s.store.Get(db, ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("记录不存在")
		}
		return err
	}
	if item.Direction != consts.DirectionIn {
		return response.BadRequest("只有收礼记录可以标记已还", "only incoming records can be marked repaid")
	}
	if err := s.store.SetStatus(db, ownerID, []uint64{id}, consts.RecordStatusRepaid); err != nil {
		return err
	}
	note := strings.TrimSpace(param.PaperNote)
	if note != "" {
		if err := s.store.SetPaperNote(db, ownerID, id, note); err != nil {
			return err
		}
	}
	s.log.Info(ctx).Uint64("id", id).Msg("record.markRepaid")
	return nil
}

func (s *RecordSrv) ParsePaperNote(ctx context.Context, param record.PaperNoteParseParam) (*record.PaperParseResult, error) {
	note := strings.TrimSpace(param.Note)
	if note == "" {
		return nil, response.BadRequest("标注内容为空", "note is empty")
	}
	result := parsePaperNote(note)
	return &result, nil
}

func (s *RecordSrv) PendingNotes(ctx context.Context, bookID uint64, page string) ([]record.Resp, error) {
	ownerID := middleware.OwnerID(ctx)
	items, err := s.store.PendingPaperNotes(s.rdb.Get(), ownerID, bookID, strings.TrimSpace(page))
	if err != nil {
		return nil, err
	}
	return s.serialize(s.rdb.Get(), ownerID, items)
}

func (s *RecordSrv) SetPageCheck(ctx context.Context, bookID uint64, page string, declaredCent int64) error {
	ownerID := middleware.OwnerID(ctx)
	page = strings.TrimSpace(page)
	if page == "" {
		return response.BadRequest("页码不能为空", "page is required")
	}
	if declaredCent < 0 {
		return response.BadRequest("本页合计不能为负数", "declared amount must be non-negative")
	}
	if _, err := s.getNormalBook(s.rdb.Get(), ownerID, bookID); err != nil {
		return err
	}
	return s.store.UpsertPageCheck(s.rdb.Get(), ownerID, bookID, page, declaredCent)
}

func (s *RecordSrv) Pages(ctx context.Context, bookID uint64) ([]book.PaperPage, error) {
	ownerID := middleware.OwnerID(ctx)
	if _, err := s.books.Get(s.rdb.Get(), ownerID, bookID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("礼簿不存在")
		}
		return nil, err
	}
	aggs, err := s.store.PageAggs(s.rdb.Get(), ownerID, bookID)
	if err != nil {
		return nil, err
	}
	checks, err := s.store.PageChecks(s.rdb.Get(), ownerID, bookID)
	if err != nil {
		return nil, err
	}
	pages := buildPages(aggs, checks)
	return pages, nil
}

func (s *RecordSrv) PageDetail(ctx context.Context, bookID uint64, page string) (*book.PaperPageDetail, error) {
	ownerID := middleware.OwnerID(ctx)
	page = strings.TrimSpace(page)
	if page == "" {
		return nil, response.BadRequest("页码不能为空", "page is required")
	}
	pages, err := s.Pages(ctx, bookID)
	if err != nil {
		return nil, err
	}
	target := book.PaperPage{Page: page}
	for i := range pages {
		if pages[i].Page == page {
			target = pages[i]
			break
		}
	}
	items, err := s.store.PageRecords(s.rdb.Get(), ownerID, bookID, page)
	if err != nil {
		return nil, err
	}
	resps, err := s.serialize(s.rdb.Get(), ownerID, items)
	if err != nil {
		return nil, err
	}
	return &book.PaperPageDetail{PaperPage: target, Records: resps}, nil
}

func (s *RecordSrv) CountAll(ctx context.Context) (int64, error) {
	return s.store.CountByOwner(s.rdb.Get(), middleware.OwnerID(ctx))
}

func (s *RecordSrv) serialize(db *gorm.DB, ownerID uint64, items []record.Record) ([]record.Resp, error) {
	if len(items) == 0 {
		return []record.Resp{}, nil
	}
	personIDs := make([]uint64, 0, len(items))
	eventIDs := make([]uint64, 0, len(items))
	bookIDs := make([]uint64, 0, len(items)*2)
	recordIDs := make([]uint64, 0, len(items))
	for i := range items {
		personIDs = append(personIDs, items[i].PersonID)
		eventIDs = append(eventIDs, items[i].EventID)
		bookIDs = append(bookIDs, items[i].BookID, items[i].OriginBookID)
		recordIDs = append(recordIDs, items[i].ID)
	}
	persons, err := s.persons.GetMany(db, ownerID, dedupeIDs(personIDs))
	if err != nil {
		return nil, err
	}
	personByID := make(map[uint64]person.Person, len(persons))
	for i := range persons {
		personByID[persons[i].ID] = persons[i]
	}
	events, err := s.events.GetMany(db, ownerID, dedupeIDs(eventIDs))
	if err != nil {
		return nil, err
	}
	eventByID := make(map[uint64]event.Event, len(events))
	for i := range events {
		eventByID[events[i].ID] = events[i]
	}
	books, err := s.books.GetMany(db, ownerID, dedupeIDs(bookIDs))
	if err != nil {
		return nil, err
	}
	bookByID := make(map[uint64]book.Book, len(books))
	for i := range books {
		bookByID[books[i].ID] = books[i]
	}
	labels, err := s.itemKindLabels(db, ownerID)
	if err != nil {
		return nil, err
	}
	repayCounts, err := s.store.LinkCountsByRepay(db, ownerID, recordIDs)
	if err != nil {
		return nil, err
	}
	originCounts, err := s.store.OriginLinkCounts(db, ownerID, recordIDs)
	if err != nil {
		return nil, err
	}

	out := make([]record.Resp, 0, len(items))
	for i := range items {
		item := items[i]
		personItem := personByID[item.PersonID]
		eventItem := eventByID[item.EventID]
		bookItem := bookByID[item.BookID]
		originBook := bookByID[item.OriginBookID]
		repayCount := repayCounts[item.ID]
		if item.Direction == consts.DirectionIn {
			repayCount = originCounts[item.ID]
		}
		originName := ""
		if item.OriginBookID != item.BookID {
			originName = originBook.Name
		}
		out = append(out, record.Resp{
			ID:             item.ID,
			PersonID:       item.PersonID,
			PersonName:     personItem.Name,
			PersonRelation: personItem.Relation,
			EventID:        item.EventID,
			EventName:      eventItem.Name,
			EventType:      eventItem.Type,
			BookID:         item.BookID,
			BookName:       bookItem.Name,
			OriginBookID:   item.OriginBookID,
			OriginBookName: originName,
			ItemKind:       item.ItemKind,
			ItemKindLabel:  labelOf(labels, item.ItemKind),
			Direction:      item.Direction,
			AmountCent:     item.AmountCent,
			GiftName:       item.GiftName,
			GiftValueCent:  item.GiftValueCent,
			TotalCent:      item.TotalCent,
			OccurredOn:     utils.DateString(item.OccurredOn),
			DatePrecision:  item.DatePrecision,
			Status:         item.Status,
			StatusLabel:    record.LabelOfStatus(item.Status),
			Remark:         item.Remark,
			PaperNote:      item.PaperNote,
			PaperPage:      item.PaperPage,
			PaperSeq:       item.PaperSeq,
			Source:         item.Source,
			RepayCount:     repayCount,
			CreatedAt:      utils.Millis(item.CreatedAt),
			UpdatedAt:      utils.Millis(item.UpdatedAt),
		})
	}
	return out, nil
}

func (s *RecordSrv) itemKindLabels(db *gorm.DB, ownerID uint64) (map[string]string, error) {
	items, err := s.dicts.List(db, ownerID, consts.DictTypeItemKind)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(items))
	for i := range items {
		labels[items[i].Code] = items[i].Label
	}
	return labels, nil
}

func (s *RecordSrv) resolvePerson(tx *gorm.DB, ownerID uint64, personID uint64, personName string) (*person.Person, error) {
	if personID > 0 {
		item, err := s.persons.Get(tx, ownerID, personID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.BadRequest("亲友不存在", "invalid person")
			}
			return nil, err
		}
		return item, nil
	}
	name := strings.TrimSpace(personName)
	if name == "" {
		return nil, response.BadRequest("请选择或填写亲友", "person is required")
	}
	matches, err := s.persons.FindExactByName(tx, ownerID, name)
	if err != nil {
		return nil, err
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}
	if len(matches) > 1 {
		candidates := make([]map[string]any, 0, len(matches))
		for i := range matches {
			candidates = append(candidates, map[string]any{
				"id":            matches[i].ID,
				"name":          matches[i].Name,
				"relation":      matches[i].Relation,
				"relationLabel": person.LabelOf(matches[i].Relation),
			})
		}
		return nil, response.BadRequestDetail(consts.ReasonPersonAmbiguous,
			"有多个同名亲友，请先选择是哪一个",
			map[string]any{"candidates": candidates})
	}
	full, initials := utils.Pinyin(name)
	created := &person.Person{
		OwnerID:      ownerID,
		Name:         name,
		NamePinyin:   full,
		NameInitials: initials,
		Relation:     consts.RelationOther,
	}
	if err := s.persons.Create(tx, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *RecordSrv) getEvent(tx *gorm.DB, ownerID uint64, id uint64) (*event.Event, error) {
	item, err := s.events.Get(tx, ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.BadRequest("事项不存在", "invalid event")
		}
		return nil, err
	}
	return item, nil
}

func (s *RecordSrv) getNormalBook(tx *gorm.DB, ownerID uint64, id uint64) (*book.Book, error) {
	item, err := s.books.Get(tx, ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.BadRequest("礼簿不存在", "invalid book")
		}
		return nil, err
	}
	if item.Status != consts.BookStatusNormal {
		return nil, response.StateConflict("这本礼簿已经合并，不能再记账")
	}
	return item, nil
}

func (s *RecordSrv) resolveItemKind(tx *gorm.DB, ownerID uint64, bookItem *book.Book, itemKind string, amountCent int64, giftName string, giftValueCent int64) (string, error) {
	itemKind = strings.TrimSpace(itemKind)
	if itemKind != "" {
		if _, err := s.dicts.GetByCode(tx, ownerID, consts.DictTypeItemKind, itemKind); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", response.BadRequest("名目不存在", "invalid item kind")
			}
			return "", err
		}
		return itemKind, nil
	}
	if strings.TrimSpace(giftName) != "" && amountCent == 0 {
		return consts.ItemKindGift, nil
	}
	if bookItem.Category == consts.DictCodeHuopao {
		return consts.ItemKindHuopao, nil
	}
	return consts.ItemKindCash, nil
}

func (s *RecordSrv) duplicateError(ownerID uint64, personItem *person.Person, eventItem *event.Event, bookItem *book.Book, itemKind string, duplicates []record.Record) error {
	labels, err := s.itemKindLabels(s.rdb.Get(), ownerID)
	if err != nil {
		labels = map[string]string{}
	}
	itemLabel := labelOf(labels, itemKind)
	message := fmt.Sprintf("%s 在「%s」的「%s · %s」下已有 %d 条记录",
		personItem.Name, eventItem.Name, bookItem.Name, itemLabel, len(duplicates))
	conflicts := make([]response.ConfirmConflict, 0, len(duplicates))
	for i := range duplicates {
		conflicts = append(conflicts, response.ConfirmConflict{
			ID: duplicates[i].ID,
			Summary: consts.DirectionLabels[duplicates[i].Direction] + " ¥" +
				amountLabel(duplicates[i].TotalCent) + " · " + utils.DateString(duplicates[i].OccurredOn),
			CreatedAt: utils.Millis(duplicates[i].CreatedAt),
		})
	}
	payload := response.ConfirmPayload{
		Kind:        consts.ConfirmDuplicateRecord,
		Title:       "可能重复录入",
		Message:     message,
		Conflicts:   conflicts,
		ConfirmText: "仍然保存",
	}
	return response.Conflict(consts.ReasonNeedConfirm, message, payload)
}

func (s *RecordSrv) hasAnyLink(db *gorm.DB, ownerID uint64, id uint64) (bool, error) {
	count, err := s.store.CountLinksByOrigin(db, ownerID, id)
	if err != nil || count > 0 {
		return count > 0, err
	}
	links, err := s.store.LinksByRepay(db, ownerID, id)
	if err != nil {
		return false, err
	}
	return len(links) > 0, nil
}

func validateAmount(amountCent int64, giftName string, giftValueCent int64) error {
	if amountCent < 0 || giftValueCent < 0 {
		return response.BadRequest("金额不能为负数", "amount must be non-negative")
	}
	if amountCent == 0 && strings.TrimSpace(giftName) == "" {
		return response.BadRequest("金额和礼品至少填一项", "amount or gift is required")
	}
	if strings.TrimSpace(giftName) == "" && giftValueCent > 0 {
		return response.BadRequest("填写礼品估价前请先填礼品名称", "gift name is required for gift value")
	}
	return nil
}

func buildPages(aggs []record.PageAgg, checks []record.PaperPageCheck) []book.PaperPage {
	byPage := make(map[string]*book.PaperPage)
	for i := range checks {
		page := checks[i].PaperPage
		item := byPage[page]
		if item == nil {
			item = &book.PaperPage{Page: page}
			byPage[page] = item
		}
		item.DeclaredCent = checks[i].DeclaredCent
		item.HasDeclared = true
	}
	for i := range aggs {
		page := aggs[i].PaperPage
		item := byPage[page]
		if item == nil {
			item = &book.PaperPage{Page: page}
			byPage[page] = item
		}
		item.RecordCent = aggs[i].RecordCent
		item.RecordCount = aggs[i].RecordCount
	}
	pages := make([]book.PaperPage, 0, len(byPage))
	for _, item := range byPage {
		item.DiffCent = item.RecordCent - item.DeclaredCent
		item.Consistent = item.HasDeclared && item.DiffCent == 0
		pages = append(pages, *item)
	}
	sort.Slice(pages, func(i, j int) bool { return naturalLess(pages[i].Page, pages[j].Page) })
	return pages
}

func naturalLess(a, b string) bool {
	ta := pageTokens(a)
	tb := pageTokens(b)
	for i := 0; i < len(ta) && i < len(tb); i++ {
		if ta[i] == tb[i] {
			continue
		}
		numA, errA := strconv.ParseInt(ta[i], 10, 64)
		numB, errB := strconv.ParseInt(tb[i], 10, 64)
		if errA == nil && errB == nil {
			return numA < numB
		}
		return ta[i] < tb[i]
	}
	return len(ta) < len(tb)
}

func pageTokens(value string) []string {
	tokens := make([]string, 0, 4)
	var current strings.Builder
	digit := false
	for _, r := range value {
		isDigit := r >= '0' && r <= '9'
		if current.Len() > 0 && isDigit != digit {
			tokens = append(tokens, current.String())
			current.Reset()
		}
		digit = isDigit
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func labelOf(labels map[string]string, code string) string {
	if label, ok := labels[code]; ok {
		return label
	}
	return code
}

func amountLabel(cent int64) string {
	if cent%100 == 0 {
		return strconv.FormatInt(cent/100, 10)
	}
	return utils.YuanString(cent)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func dedupeIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
