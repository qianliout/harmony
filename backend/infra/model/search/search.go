// Package search 定义全局搜索的分组结果。
package search

import (
	"harmony/infra/model/record"
)

type PersonHit struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Relation      string `json:"relation"`
	RelationLabel string `json:"relationLabel"`
	RecordCount   int64  `json:"recordCount"`
}

type EventHit struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	TypeLabel string `json:"typeLabel"`
	EventDate string `json:"eventDate"`
	Host      string `json:"host"`
}

type BookHit struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	CategoryLabel string `json:"categoryLabel"`
	RecordCount   int64  `json:"recordCount"`
}

type RecordHit struct {
	record.Resp
	MatchField string `json:"matchField"`
}

type Group[T any] struct {
	Total int64 `json:"total"`
	Items []T   `json:"items"`
}

type Result struct {
	Intent  string                  `json:"intent"`
	Query   string                  `json:"query"`
	Persons Group[PersonHit]        `json:"persons"`
	Events  Group[EventHit]         `json:"events"`
	Books   Group[BookHit]          `json:"books"`
	Records Group[RecordHit]        `json:"records"`
}
