// Package view 定义只读聚合视图的出参：人时间线、事项汇总、待还清单。
package view

import (
	"harmony/infra/model/event"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
)

type Part struct {
	Label string `json:"label"`
	Cent  int64  `json:"cent"`
}

type EventBrief struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	EventDate string `json:"eventDate"`
	Host      string `json:"host"`
	Location  string `json:"location"`
}

type TimelineGroup struct {
	Event   EventBrief    `json:"event"`
	InCent  int64         `json:"inCent"`
	OutCent int64         `json:"outCent"`
	Parts   []Part        `json:"parts"`
	Records []record.Resp `json:"records"`
}

type Timeline struct {
	Groups []TimelineGroup `json:"groups"`
}

type PersonDetailLine struct {
	BookName      string   `json:"bookName"`
	ItemKindLabel string   `json:"itemKindLabel"`
	Direction     string   `json:"direction"`
	Cent          int64    `json:"cent"`
	RecordIDs     []uint64 `json:"recordIds"`
}

type EventPerson struct {
	Person       person.Ref         `json:"person"`
	InCent       int64              `json:"inCent"`
	OutCent      int64              `json:"outCent"`
	PendingCount int64              `json:"pendingCount"`
	Details      []PersonDetailLine `json:"details"`
}

type EventSummary struct {
	Event   EventBrief     `json:"event"`
	Summary event.Summary  `json:"summary"`
	Persons []EventPerson  `json:"persons"`
}

type PendingPerson struct {
	Person     person.Ref `json:"person"`
	Count      int64      `json:"count"`
	TotalCent  int64      `json:"totalCent"`
	Earliest   string     `json:"earliest"`
}

type PendingList struct {
	Persons     []PendingPerson `json:"persons"`
	PersonCount int64           `json:"personCount"`
	TotalCount  int64           `json:"totalCount"`
	TotalCent   int64           `json:"totalCent"`
}
