package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"harmony/component/book/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/book"
	"harmony/infra/model/dict"
	"harmony/infra/model/event"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type DictStore interface {
	GetByCode(db *gorm.DB, ownerID uint64, dictType, code string) (*dict.Item, error)
	List(db *gorm.DB, ownerID uint64, dictType string) ([]dict.Item, error)
}

type EventStore interface {
	Get(db *gorm.DB, ownerID uint64, id uint64) (*event.Event, error)
	GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]event.Event, error)
}

type BookSrv struct {
	rdb     *rdb.RDB
	store   *store.Store
	dicts   DictStore
	events  EventStore
	log     *logger.Logger
}

func NewBookSrv(database *rdb.RDB, bookStore *store.Store, dictStore DictStore, eventStore EventStore) *BookSrv {
	return &BookSrv{
		rdb:    database,
		store:  bookStore,
		dicts:  dictStore,
		events: eventStore,
		log:    logger.New(logger.WithModule("book"), logger.WithSubModule("service")),
	}
}

func (s *BookSrv) List(ctx context.Context, status string) ([]book.Resp, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = consts.BookStatusNormal
	}
	if !consts.ValidBookStatus(status) {
		return nil, response.BadRequest("礼簿状态不正确", "invalid book status")
	}
	ownerID := middleware.OwnerID(ctx)
	items, err := s.store.List(s.rdb.Get(), ownerID, status)
	if err != nil {
		return nil, err
	}
	return s.serializeMany(ctx, ownerID, items)
}

func (s *BookSrv) Detail(ctx context.Context, id uint64) (*book.Detail, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("礼簿不存在")
		}
		return nil, err
	}
	resps, err := s.serializeMany(ctx, ownerID, []book.Book{*item})
	if err != nil {
		return nil, err
	}
	parts, err := s.store.ByItem(s.rdb.Get(), ownerID, id)
	if err != nil {
		return nil, err
	}
	labels, err := s.itemKindLabels(ownerID)
	if err != nil {
		return nil, err
	}
	for i := range parts {
		parts[i].ItemKindLabel = labelOf(labels, parts[i].ItemKind)
	}
	return &book.Detail{Resp: resps[0], ByItem: parts}, nil
}

func (s *BookSrv) Create(ctx context.Context, param book.CreateParam) (*book.Detail, error) {
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil, response.BadRequest("礼簿名称不能为空", "book name is required")
	}
	category := strings.TrimSpace(param.Category)
	if category == "" {
		category = consts.DictCodeRenqing
	}
	ownerID := middleware.OwnerID(ctx)
	if _, err := s.dicts.GetByCode(s.rdb.Get(), ownerID, consts.DictTypeBookCategory, category); err != nil {
		return nil, response.BadRequest("礼簿类别不存在", "invalid book category")
	}
	if param.DefaultEventID > 0 {
		if _, err := s.events.Get(s.rdb.Get(), ownerID, param.DefaultEventID); err != nil {
			return nil, response.BadRequest("默认事项不存在", "invalid default event")
		}
	}
	item := &book.Book{
		OwnerID:     ownerID,
		Name:        name,
		Category:    category,
		Description: strings.TrimSpace(param.Description),
		Status:      consts.BookStatusNormal,
	}
	if param.DefaultEventID > 0 {
		item.DefaultEventID = &param.DefaultEventID
	}
	if err := s.store.Create(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("id", item.ID).Msg("book.create")
	return s.Detail(ctx, item.ID)
}

func (s *BookSrv) Update(ctx context.Context, id uint64, param book.UpdateParam) (*book.Detail, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("礼簿不存在")
		}
		return nil, err
	}
	if item.Status == consts.BookStatusMerged {
		return nil, response.StateConflict("这本礼簿已经合并，不能再修改")
	}
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil, response.BadRequest("礼簿名称不能为空", "book name is required")
	}
	category := strings.TrimSpace(param.Category)
	if category == "" {
		category = item.Category
	}
	if _, err := s.dicts.GetByCode(s.rdb.Get(), ownerID, consts.DictTypeBookCategory, category); err != nil {
		return nil, response.BadRequest("礼簿类别不存在", "invalid book category")
	}
	item.Name = name
	item.Category = category
	item.Description = strings.TrimSpace(param.Description)
	if param.DefaultEventID != nil {
		if *param.DefaultEventID > 0 {
			if _, err := s.events.Get(s.rdb.Get(), ownerID, *param.DefaultEventID); err != nil {
				return nil, response.BadRequest("默认事项不存在", "invalid default event")
			}
			item.DefaultEventID = param.DefaultEventID
		} else {
			item.DefaultEventID = nil
		}
	}
	if err := s.store.Update(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("book.update")
	return s.Detail(ctx, id)
}

func (s *BookSrv) Delete(ctx context.Context, id uint64) error {
	ownerID := middleware.OwnerID(ctx)
	if _, err := s.store.Get(s.rdb.Get(), ownerID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("礼簿不存在")
		}
		return err
	}
	count, err := s.store.CountRecords(s.rdb.Get(), ownerID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return response.HasReference("这本礼簿下还有记录，先处理记录再删除", count)
	}
	if err := s.store.Delete(s.rdb.Get(), ownerID, id); err != nil {
		return err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("book.delete")
	return nil
}

func (s *BookSrv) MergePreview(ctx context.Context, param book.MergeParam) (*book.MergePreview, error) {
	ownerID := middleware.OwnerID(ctx)
	sources, target, err := s.validateMerge(ownerID, param)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(sources))
	cross := false
	categories := make([]string, 0, len(sources)+1)
	categories = append(categories, target.Category)
	for i := range sources {
		ids = append(ids, sources[i].ID)
		categories = append(categories, sources[i].Category)
		if sources[i].Category != target.Category {
			cross = true
		}
	}
	count, err := s.store.CountRecordsByBooks(s.rdb.Get(), ownerID, ids)
	if err != nil {
		return nil, err
	}
	itemCounts, err := s.store.ItemCountsByBooks(s.rdb.Get(), ownerID, ids)
	if err != nil {
		return nil, err
	}
	labels, err := s.itemKindLabels(ownerID)
	if err != nil {
		return nil, err
	}
	for i := range itemCounts {
		itemCounts[i].ItemKindLabel = labelOf(labels, itemCounts[i].ItemKind)
	}
	sourceResps, err := s.serializeMany(ctx, ownerID, sources)
	if err != nil {
		return nil, err
	}
	targetResps, err := s.serializeMany(ctx, ownerID, []book.Book{*target})
	if err != nil {
		return nil, err
	}
	return &book.MergePreview{
		Target:        targetResps[0],
		Sources:       sourceResps,
		RecordCount:   count,
		CrossCategory: cross,
		Categories:    categories,
		ItemCounts:    itemCounts,
	}, nil
}

func (s *BookSrv) Merge(ctx context.Context, param book.MergeParam) (*book.MergeLog, error) {
	ownerID := middleware.OwnerID(ctx)
	sources, target, err := s.validateMerge(ownerID, param)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(sources))
	categories := make([]string, 0, len(sources))
	snapshots := make([]book.MergeSourceSnapshot, 0, len(sources))
	cross := false
	agg, err := s.store.Aggregates(s.rdb.Get(), ownerID, idsOfBooks(sources))
	if err != nil {
		return nil, err
	}
	for i := range sources {
		ids = append(ids, sources[i].ID)
		categories = append(categories, sources[i].Category)
		if sources[i].Category != target.Category {
			cross = true
		}
		snapshots = append(snapshots, book.MergeSourceSnapshot{
			ID:          sources[i].ID,
			Name:        sources[i].Name,
			Category:    sources[i].Category,
			RecordCount: agg[sources[i].ID].RecordCount,
		})
	}
	snapshotJSON, err := json.Marshal(snapshots)
	if err != nil {
		return nil, err
	}

	var logRow book.MergeLogRow
	err = s.rdb.Transaction(func(tx *gorm.DB) error {
		locked, err := s.store.GetBooksInTx(tx, ownerID, append(ids, target.ID))
		if err != nil {
			return err
		}
		lockedByID := make(map[uint64]book.Book, len(locked))
		for i := range locked {
			lockedByID[locked[i].ID] = locked[i]
		}
		lockedTarget, ok := lockedByID[target.ID]
		if !ok || lockedTarget.Status != consts.BookStatusNormal {
			return response.StateConflict("目标礼簿状态已变化，请刷新后重试")
		}
		for _, id := range ids {
			item, ok := lockedByID[id]
			if !ok || item.Status != consts.BookStatusNormal {
				return response.StateConflict("来源礼簿状态已变化，请刷新后重试")
			}
		}
		moved, err := s.store.MoveRecords(tx, ownerID, ids, target.ID)
		if err != nil {
			return err
		}
		if err := s.store.MovePageChecks(tx, ownerID, ids, target.ID); err != nil {
			return err
		}
		if err := s.store.RepointMergedInto(tx, ownerID, ids, target.ID); err != nil {
			return err
		}
		if err := s.store.MarkMerged(tx, ownerID, ids, target.ID); err != nil {
			return err
		}
		logRow = book.MergeLogRow{
			OwnerID:       ownerID,
			TargetBookID:  target.ID,
			SourceBooks:   string(snapshotJSON),
			RecordCount:   int64(moved),
			CrossCategory: cross,
		}
		return s.store.CreateMergeLog(tx, &logRow)
	})
	if err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("target", target.ID).Int("sources", len(ids)).Int64("records", logRow.RecordCount).Msg("book.merge")

	return &book.MergeLog{
		ID:            logRow.ID,
		TargetBookID:  target.ID,
		TargetName:    target.Name,
		Sources:       snapshots,
		RecordCount:   logRow.RecordCount,
		CrossCategory: cross,
		CreatedAt:     utils.Millis(logRow.CreatedAt),
	}, nil
}

func (s *BookSrv) MergeLogs(ctx context.Context) ([]book.MergeLog, error) {
	ownerID := middleware.OwnerID(ctx)
	rows, err := s.store.ListMergeLogs(s.rdb.Get(), ownerID)
	if err != nil {
		return nil, err
	}
	targetIDs := make([]uint64, 0, len(rows))
	for i := range rows {
		targetIDs = append(targetIDs, rows[i].TargetBookID)
	}
	targets, err := s.store.GetMany(s.rdb.Get(), ownerID, targetIDs)
	if err != nil {
		return nil, err
	}
	names := make(map[uint64]string, len(targets))
	for i := range targets {
		names[targets[i].ID] = targets[i].Name
	}
	out := make([]book.MergeLog, 0, len(rows))
	for i := range rows {
		sources := make([]book.MergeSourceSnapshot, 0)
		if err := json.Unmarshal([]byte(rows[i].SourceBooks), &sources); err != nil {
			sources = []book.MergeSourceSnapshot{}
		}
		out = append(out, book.MergeLog{
			ID:            rows[i].ID,
			TargetBookID:  rows[i].TargetBookID,
			TargetName:    names[rows[i].TargetBookID],
			Sources:       sources,
			RecordCount:   rows[i].RecordCount,
			CrossCategory: rows[i].CrossCategory,
			CreatedAt:     utils.Millis(rows[i].CreatedAt),
		})
	}
	return out, nil
}

func (s *BookSrv) validateMerge(ownerID uint64, param book.MergeParam) ([]book.Book, *book.Book, error) {
	if param.TargetID == 0 {
		return nil, nil, response.BadRequest("请选择目标礼簿", "target book is required")
	}
	sourceIDs := dedupeIDs(param.SourceIDs)
	if len(sourceIDs) == 0 {
		return nil, nil, response.BadRequest("请选择要合并的礼簿", "source books are required")
	}
	for _, id := range sourceIDs {
		if id == param.TargetID {
			return nil, nil, response.BadRequest("目标礼簿不能同时作为来源", "target book cannot be a source")
		}
	}
	target, err := s.store.Get(s.rdb.Get(), ownerID, param.TargetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, response.NotFound("目标礼簿不存在")
		}
		return nil, nil, err
	}
	if target.Status != consts.BookStatusNormal {
		return nil, nil, response.StateConflict("目标礼簿已合并，请选择正常礼簿")
	}
	sources, err := s.store.GetMany(s.rdb.Get(), ownerID, sourceIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(sources) != len(sourceIDs) {
		return nil, nil, response.NotFound("部分来源礼簿不存在")
	}
	for i := range sources {
		if sources[i].Status != consts.BookStatusNormal {
			return nil, nil, response.StateConflict("来源礼簿中有已合并的礼簿，请刷新后重试")
		}
	}
	return sources, target, nil
}

func (s *BookSrv) serializeMany(ctx context.Context, ownerID uint64, items []book.Book) ([]book.Resp, error) {
	bookIDs := make([]uint64, 0, len(items))
	eventIDs := make([]uint64, 0, len(items))
	mergedIDs := make([]uint64, 0, len(items))
	for i := range items {
		bookIDs = append(bookIDs, items[i].ID)
		if items[i].DefaultEventID != nil {
			eventIDs = append(eventIDs, *items[i].DefaultEventID)
		}
		if items[i].MergedIntoID != nil {
			mergedIDs = append(mergedIDs, *items[i].MergedIntoID)
		}
	}
	agg, err := s.store.Aggregates(s.rdb.Get(), ownerID, bookIDs)
	if err != nil {
		return nil, err
	}
	eventNames, err := s.eventNames(ownerID, eventIDs)
	if err != nil {
		return nil, err
	}
	merged, err := s.store.GetMany(s.rdb.Get(), ownerID, mergedIDs)
	if err != nil {
		return nil, err
	}
	mergedNames := make(map[uint64]string, len(merged))
	for i := range merged {
		mergedNames[merged[i].ID] = merged[i].Name
	}
	dicts, err := s.dicts.List(s.rdb.Get(), ownerID, consts.DictTypeBookCategory)
	if err != nil {
		return nil, err
	}
	categoryByCode := make(map[string]dict.Item, len(dicts))
	for i := range dicts {
		categoryByCode[dicts[i].Code] = dicts[i]
	}
	out := make([]book.Resp, 0, len(items))
	for i := range items {
		item := items[i]
		category := categoryByCode[item.Category]
		resp := book.Resp{
			ID:            item.ID,
			Name:          item.Name,
			Category:      item.Category,
			CategoryLabel: category.Label,
			CategoryColor: category.Color,
			Description:   item.Description,
			Status:        item.Status,
			MergedAt:      utils.MillisPtr(item.MergedAt),
			RecordCount:   agg[item.ID].RecordCount,
			InCent:        agg[item.ID].InCent,
			OutCent:       agg[item.ID].OutCent,
			PendingCount:  agg[item.ID].PendingCount,
			CreatedAt:     utils.Millis(item.CreatedAt),
			UpdatedAt:     utils.Millis(item.UpdatedAt),
		}
		if item.DefaultEventID != nil {
			resp.DefaultEventID = *item.DefaultEventID
			resp.DefaultEventName = eventNames[*item.DefaultEventID]
		}
		if item.MergedIntoID != nil {
			resp.MergedIntoID = *item.MergedIntoID
			resp.MergedIntoName = mergedNames[*item.MergedIntoID]
		}
		out = append(out, resp)
	}
	return out, nil
}

func (s *BookSrv) eventNames(ownerID uint64, ids []uint64) (map[uint64]string, error) {
	names := make(map[uint64]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	events, err := s.events.GetMany(s.rdb.Get(), ownerID, dedupeIDs(ids))
	if err != nil {
		return nil, err
	}
	for i := range events {
		names[events[i].ID] = events[i].Name
	}
	return names, nil
}

func (s *BookSrv) itemKindLabels(ownerID uint64) (map[string]string, error) {
	items, err := s.dicts.List(s.rdb.Get(), ownerID, consts.DictTypeItemKind)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(items))
	for i := range items {
		labels[items[i].Code] = items[i].Label
	}
	return labels, nil
}

func labelOf(labels map[string]string, code string) string {
	if label, ok := labels[code]; ok {
		return label
	}
	return code
}

func idsOfBooks(items []book.Book) []uint64 {
	out := make([]uint64, 0, len(items))
	for i := range items {
		out = append(out, items[i].ID)
	}
	return out
}

func dedupeIDs(ids []uint64) []uint64 {
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
