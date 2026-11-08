package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	bookstore "harmony/component/book/store"
	dictstore "harmony/component/dict/store"
	personstore "harmony/component/person/store"
	"harmony/component/stats/store"
	viewservice "harmony/component/view/service"
	"harmony/infra/consts"
	"harmony/infra/middleware"
	"harmony/infra/model/book"
	"harmony/infra/model/dict"
	"harmony/infra/model/person"
	"harmony/infra/model/stats"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type StatsSrv struct {
	rdb    *rdb.RDB
	store  *store.Store
	view   *viewservice.ViewSrv
	persons *personstore.Store
	books  *bookstore.Store
	dicts  *dictstore.Store
}

func NewStatsSrv(
	database *rdb.RDB,
	statsStore *store.Store,
	viewSrv *viewservice.ViewSrv,
	personStore *personstore.Store,
	bookStore *bookstore.Store,
	dictStore *dictstore.Store,
) *StatsSrv {
	return &StatsSrv{
		rdb:     database,
		store:   statsStore,
		view:    viewSrv,
		persons: personStore,
		books:   bookStore,
		dicts:   dictStore,
	}
}

func (s *StatsSrv) Years(ctx context.Context) ([]int64, error) {
	return s.store.Years(s.rdb.Get(), middleware.OwnerID(ctx))
}

func (s *StatsSrv) Year(ctx context.Context, year int64) (*stats.Year, error) {
	if year <= 0 {
		year = int64(time.Now().Year())
	}
	ownerID := middleware.OwnerID(ctx)
	startTime, endTime := utils.YearRange(year)
	start, end := utils.DateString(startTime), utils.DateString(endTime)
	totals, err := s.store.Totals(s.rdb.Get(), ownerID, start, end)
	if err != nil {
		return nil, err
	}
	monthlyRows, err := s.store.Monthly(s.rdb.Get(), ownerID, start, end)
	if err != nil {
		return nil, err
	}
	categoryRows, err := s.store.ByBookCategory(s.rdb.Get(), ownerID, start, end)
	if err != nil {
		return nil, err
	}
	itemRows, err := s.store.ByItem(s.rdb.Get(), ownerID, start, end)
	if err != nil {
		return nil, err
	}
	typeRows, err := s.store.ByEventType(s.rdb.Get(), ownerID, start, end)
	if err != nil {
		return nil, err
	}
	personRows, err := s.store.TopPersons(s.rdb.Get(), ownerID, start, end, 10)
	if err != nil {
		return nil, err
	}
	bookRows, err := s.store.ByBook(s.rdb.Get(), ownerID, start, end)
	if err != nil {
		return nil, err
	}
	itemLabels, err := s.itemKindLabels(ownerID)
	if err != nil {
		return nil, err
	}
	categoryLabels, err := s.categoryLabels(ownerID)
	if err != nil {
		return nil, err
	}

	monthly := make([]stats.MonthStat, 12)
	for i := range monthly {
		monthly[i].Month = i + 1
	}
	for _, row := range monthlyRows {
		if row.Month < 1 || row.Month > 12 {
			continue
		}
		entry := &monthly[row.Month-1]
		entry.RecordCount += row.RecordCount
		if row.Direction == consts.DirectionIn {
			entry.InCent += row.TotalCent
		} else {
			entry.OutCent += row.TotalCent
		}
	}

	byCategory := make([]stats.BookCategoryStat, 0, len(categoryRows))
	for _, row := range categoryRows {
		byCategory = append(byCategory, stats.BookCategoryStat{
			Category:    row.Category,
			Label:       labelOr(categoryLabels, row.Category),
			InCent:      row.InCent,
			OutCent:     row.OutCent,
			Cent:        row.TotalCent,
			RecordCount: row.RecordCount,
		})
	}
	byItem := make([]stats.KindStat, 0, len(itemRows))
	for _, row := range itemRows {
		byItem = append(byItem, stats.KindStat{
			ItemKind:      row.ItemKind,
			ItemKindLabel: labelOr(itemLabels, row.ItemKind),
			Direction:     row.Direction,
			Cent:          row.TotalCent,
			RecordCount:   row.RecordCount,
		})
	}
	byType := make([]stats.EventTypeStat, 0, len(typeRows))
	for _, row := range typeRows {
		byType = append(byType, stats.EventTypeStat{
			Type:        row.Type,
			TypeLabel:   labelOr(consts.EventTypeLabels, row.Type),
			Cent:        row.TotalCent,
			RecordCount: row.RecordCount,
			EventCount:  row.EventCount,
		})
	}
	topPersons := make([]stats.PersonStat, 0, len(personRows))
	for _, row := range personRows {
		topPersons = append(topPersons, stats.PersonStat{
			Person: person.Ref{
				ID:            row.PersonID,
				Name:          row.Name,
				Relation:      row.Relation,
				RelationLabel: person.LabelOf(row.Relation),
			},
			InCent:      row.InCent,
			OutCent:     row.OutCent,
			NetCent:     row.InCent - row.OutCent,
			RecordCount: row.RecordCount,
		})
	}
	byBook := make([]stats.BookStat, 0, len(bookRows))
	for _, row := range bookRows {
		byBook = append(byBook, stats.BookStat{
			Book: book.Resp{
				ID:            row.BookID,
				Name:          row.Name,
				Category:      row.Category,
				CategoryLabel: labelOr(categoryLabels, row.Category),
			},
			InCent:      row.InCent,
			OutCent:     row.OutCent,
			RecordCount: row.RecordCount,
		})
	}
	return &stats.Year{
		Year:        year,
		InCent:      totals.InCent,
		OutCent:     totals.OutCent,
		NetCent:     totals.InCent - totals.OutCent,
		RecordCount: totals.RecordCount,
		PersonCount: totals.PersonCount,
		Monthly:        monthly,
		ByBookCategory: byCategory,
		ByItem:         byItem,
		ByEventType: byType,
		TopPersons:  topPersons,
		ByBook:      byBook,
	}, nil
}

func (s *StatsSrv) Event(ctx context.Context, eventID uint64) (*stats.EventStats, error) {
	summary, err := s.view.EventSummary(ctx, eventID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.KindByEvent(s.rdb.Get(), middleware.OwnerID(ctx), eventID)
	if err != nil {
		return nil, err
	}
	labels, err := s.itemKindLabels(middleware.OwnerID(ctx))
	if err != nil {
		return nil, err
	}
	byBook := make([]stats.KindStat, 0, len(rows))
	for _, row := range rows {
		byBook = append(byBook, stats.KindStat{
			ItemKind:      row.ItemKind,
			ItemKindLabel: labelOr(labels, row.ItemKind),
			Direction:     row.Direction,
			Cent:          row.TotalCent,
			RecordCount:   row.RecordCount,
		})
	}
	return &stats.EventStats{Summary: *summary, ByBook: byBook}, nil
}

func (s *StatsSrv) Person(ctx context.Context, personID uint64) (*stats.PersonStat, error) {
	ownerID := middleware.OwnerID(ctx)
	personItem, err := s.persons.Get(s.rdb.Get(), ownerID, personID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("亲友不存在")
		}
		return nil, err
	}
	summaries, err := s.persons.Summaries(s.rdb.Get(), ownerID, []uint64{personID})
	if err != nil {
		return nil, err
	}
	summary := summaries[personID]
	return &stats.PersonStat{
		Person:      personItem.Ref(),
		InCent:      summary.InCent,
		OutCent:     summary.OutCent,
		NetCent:     summary.NetCent,
		RecordCount: summary.RecordCount,
	}, nil
}

func (s *StatsSrv) Book(ctx context.Context, bookID uint64) (*stats.BookStats, error) {
	ownerID := middleware.OwnerID(ctx)
	bookItem, err := s.books.Get(s.rdb.Get(), ownerID, bookID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("礼簿不存在")
		}
		return nil, err
	}
	agg, err := s.books.Aggregates(s.rdb.Get(), ownerID, []uint64{bookID})
	if err != nil {
		return nil, err
	}
	parts, err := s.books.ByItem(s.rdb.Get(), ownerID, bookID)
	if err != nil {
		return nil, err
	}
	labels, err := s.itemKindLabels(ownerID)
	if err != nil {
		return nil, err
	}
	byItem := make([]stats.KindStat, 0, len(parts))
	for _, part := range parts {
		byItem = append(byItem, stats.KindStat{
			ItemKind:      part.ItemKind,
			ItemKindLabel: labelOr(labels, part.ItemKind),
			Direction:     part.Direction,
			Cent:          part.Cent,
			RecordCount:   part.RecordCount,
		})
	}
	summary := agg[bookID]
	categoryColor := ""
	if colors, colorErr := s.categoryColors(ownerID); colorErr == nil {
		categoryColor = colors[bookItem.Category]
	}
	return &stats.BookStats{
		Book: book.Resp{
			ID:            bookItem.ID,
			Name:          bookItem.Name,
			Category:      bookItem.Category,
			CategoryLabel: labelOr(mustCategoryLabels(s, ownerID), bookItem.Category),
			CategoryColor: categoryColor,
			Status:        bookItem.Status,
			RecordCount:   summary.RecordCount,
			InCent:        summary.InCent,
			OutCent:       summary.OutCent,
			PendingCount:  summary.PendingCount,
			CreatedAt:     utils.Millis(bookItem.CreatedAt),
			UpdatedAt:     utils.Millis(bookItem.UpdatedAt),
		},
		InCent:       summary.InCent,
		OutCent:      summary.OutCent,
		PendingCount: summary.PendingCount,
		RecordCount:  summary.RecordCount,
		ByItem:       byItem,
	}, nil
}

func (s *StatsSrv) itemKindLabels(ownerID uint64) (map[string]string, error) {
	items, err := s.dicts.List(s.rdb.Get(), ownerID, consts.DictTypeItemKind)
	if err != nil {
		return nil, err
	}
	return labelMap(items), nil
}

func (s *StatsSrv) categoryLabels(ownerID uint64) (map[string]string, error) {
	items, err := s.dicts.List(s.rdb.Get(), ownerID, consts.DictTypeBookCategory)
	if err != nil {
		return nil, err
	}
	return labelMap(items), nil
}

func (s *StatsSrv) categoryColors(ownerID uint64) (map[string]string, error) {
	items, err := s.dicts.List(s.rdb.Get(), ownerID, consts.DictTypeBookCategory)
	if err != nil {
		return nil, err
	}
	colors := make(map[string]string, len(items))
	for i := range items {
		colors[items[i].Code] = items[i].Color
	}
	return colors, nil
}

func mustCategoryLabels(s *StatsSrv, ownerID uint64) map[string]string {
	labels, err := s.categoryLabels(ownerID)
	if err != nil {
		return map[string]string{}
	}
	return labels
}

func labelMap(items []dict.Item) map[string]string {
	out := make(map[string]string, len(items))
	for i := range items {
		out[items[i].Code] = items[i].Label
	}
	return out
}

func labelOr(labels map[string]string, code string) string {
	if label, ok := labels[code]; ok {
		return label
	}
	return code
}
