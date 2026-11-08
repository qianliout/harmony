// Package stats 定义年、事项、人、礼簿的统计出参。
package stats

import (
	"harmony/infra/model/book"
	"harmony/infra/model/person"
	"harmony/infra/model/view"
)

type MonthStat struct {
	Month     int    `json:"month"`
	InCent    int64  `json:"inCent"`
	OutCent   int64  `json:"outCent"`
	RecordCount int64 `json:"recordCount"`
}

type KindStat struct {
	ItemKind      string `json:"itemKind"`
	ItemKindLabel string `json:"itemKindLabel"`
	Direction     string `json:"direction"`
	Cent          int64  `json:"cent"`
	RecordCount   int64  `json:"recordCount"`
}

type EventTypeStat struct {
	Type        string `json:"type"`
	TypeLabel   string `json:"typeLabel"`
	Cent        int64  `json:"cent"`
	RecordCount int64  `json:"recordCount"`
	EventCount  int64  `json:"eventCount"`
}

type PersonStat struct {
	Person    person.Ref `json:"person"`
	InCent    int64      `json:"inCent"`
	OutCent   int64      `json:"outCent"`
	NetCent   int64      `json:"netCent"`
	RecordCount int64    `json:"recordCount"`
}

type BookStat struct {
	Book      book.Resp `json:"book"`
	InCent    int64     `json:"inCent"`
	OutCent   int64     `json:"outCent"`
	RecordCount int64   `json:"recordCount"`
}

type Year struct {
	Year         int64          `json:"year"`
	InCent       int64          `json:"inCent"`
	OutCent      int64          `json:"outCent"`
	NetCent      int64          `json:"netCent"`
	RecordCount  int64          `json:"recordCount"`
	PersonCount  int64          `json:"personCount"`
	Monthly      []MonthStat    `json:"monthly"`
	ByItem       []KindStat     `json:"byItem"`
	ByEventType  []EventTypeStat `json:"byEventType"`
	TopPersons   []PersonStat   `json:"topPersons"`
	ByBook       []BookStat     `json:"byBook"`
}

type EventStats struct {
	Summary view.EventSummary `json:"summary"`
	ByBook  []KindStat        `json:"byBook"`
}

type BookStats struct {
	Book        book.Resp `json:"book"`
	InCent      int64     `json:"inCent"`
	OutCent     int64     `json:"outCent"`
	PendingCount int64    `json:"pendingCount"`
	RecordCount int64     `json:"recordCount"`
	ByItem      []KindStat `json:"byItem"`
}
