package service

import (
	"context"
	"strings"

	recordservice "harmony/component/record/service"
	"harmony/component/search/store"
	"harmony/infra/consts"
	"harmony/infra/middleware"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/record"
	"harmony/infra/model/search"
	"harmony/infra/rdb"
)

type SearchSrv struct {
	rdb     *rdb.RDB
	store   *store.Store
	records *recordservice.RecordSrv
}

func NewSearchSrv(database *rdb.RDB, searchStore *store.Store, recordSrv *recordservice.RecordSrv) *SearchSrv {
	return &SearchSrv{
		rdb:     database,
		store:   searchStore,
		records: recordSrv,
	}
}

func (s *SearchSrv) Search(ctx context.Context, query string, limit int) (*search.Result, error) {
	query = strings.TrimSpace(query)
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	ownerID := middleware.OwnerID(ctx)
	intent := record.ParseIntent(query)
	result := &search.Result{
		Intent: intent.Kind,
		Query:  query,
		Persons: search.Group[search.PersonHit]{Items: []search.PersonHit{}},
		Events:  search.Group[search.EventHit]{Items: []search.EventHit{}},
		Books:   search.Group[search.BookHit]{Items: []search.BookHit{}},
		Records: search.Group[search.RecordHit]{Items: []search.RecordHit{}},
	}
	if query == "" {
		return result, nil
	}

	switch intent.Kind {
	case record.IntentText:
		persons, total, err := s.store.Persons(s.rdb.Get(), ownerID, query, limit)
		if err != nil {
			return nil, err
		}
		store.FillPersonLabels(persons)
		result.Persons = search.Group[search.PersonHit]{Total: total, Items: persons}
		events, eventTotal, err := s.store.Events(s.rdb.Get(), ownerID, query, limit)
		if err != nil {
			return nil, err
		}
		fillEventLabels(events)
		result.Events = search.Group[search.EventHit]{Total: eventTotal, Items: events}
		books, bookTotal, err := s.store.Books(s.rdb.Get(), ownerID, query, limit)
		if err != nil {
			return nil, err
		}
		fillBookLabels(books)
		result.Books = search.Group[search.BookHit]{Total: bookTotal, Items: books}
	case record.IntentYear, record.IntentYearAmount:
		events, total, err := s.store.EventsByYear(s.rdb.Get(), ownerID, intent.Year, limit)
		if err != nil {
			return nil, err
		}
		fillEventLabels(events)
		result.Events = search.Group[search.EventHit]{Total: total, Items: events}
	}

	items, total, err := s.records.List(ctx, record.Query{Keyword: query, Intent: intent},
		&modelfilter.Filter{Offset: 0, Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	hits := make([]search.RecordHit, 0, len(items))
	for i := range items {
		hits = append(hits, search.RecordHit{
			Resp:       items[i],
			MatchField: record.MatchField(&record.Record{
				PaperNote: items[i].PaperNote,
				GiftName:  items[i].GiftName,
				Remark:    items[i].Remark,
			}, intent),
		})
	}
	result.Records = search.Group[search.RecordHit]{Total: total, Items: hits}
	return result, nil
}

func (s *SearchSrv) Records(ctx context.Context, query string, page int64, pageSize int64) ([]search.RecordHit, int64, error) {
	query = strings.TrimSpace(query)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > consts.MaxPageSize {
		pageSize = consts.DefaultPageSize
	}
	intent := record.ParseIntent(query)
	items, total, err := s.records.List(ctx, record.Query{Keyword: query, Intent: intent},
		&modelfilter.Filter{Offset: (page - 1) * pageSize, Limit: pageSize})
	if err != nil {
		return nil, 0, err
	}
	hits := make([]search.RecordHit, 0, len(items))
	for i := range items {
		hits = append(hits, search.RecordHit{
			Resp:       items[i],
			MatchField: record.MatchField(&record.Record{
				PaperNote: items[i].PaperNote,
				GiftName:  items[i].GiftName,
				Remark:    items[i].Remark,
			}, intent),
		})
	}
	return hits, total, nil
}

func fillEventLabels(items []search.EventHit) {
	for i := range items {
		if label, ok := consts.EventTypeLabels[items[i].Type]; ok {
			items[i].TypeLabel = label
		} else {
			items[i].TypeLabel = consts.EventTypeLabels[consts.EventTypeOther]
		}
	}
}

func fillBookLabels(items []search.BookHit) {
	for i := range items {
		if items[i].CategoryLabel == "" {
			items[i].CategoryLabel = items[i].Category
		}
	}
}
