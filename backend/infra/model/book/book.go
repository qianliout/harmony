// Package book 定义礼簿模型、合并日志与按页核对出参。
package book

import (
	"time"

	"harmony/infra/model/record"
)

type Book struct {
	ID             uint64     `gorm:"primaryKey" json:"id"`
	OwnerID        uint64     `gorm:"column:owner_id" json:"ownerId"`
	Name           string     `gorm:"column:name" json:"name"`
	Category       string     `gorm:"column:category" json:"category"`
	DefaultEventID *uint64    `gorm:"column:default_event_id" json:"defaultEventId"`
	Description    string     `gorm:"column:description" json:"description"`
	Status         string     `gorm:"column:status" json:"status"`
	MergedIntoID   *uint64    `gorm:"column:merged_into_id" json:"mergedIntoId"`
	MergedAt       *time.Time `gorm:"column:merged_at" json:"-"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"-"`
}

func (*Book) TableName() string { return "book" }

type Resp struct {
	ID              uint64 `json:"id"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	CategoryLabel   string `json:"categoryLabel"`
	CategoryColor   string `json:"categoryColor"`
	DefaultEventID  uint64 `json:"defaultEventId"`
	DefaultEventName string `json:"defaultEventName"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	MergedIntoID    uint64 `json:"mergedIntoId"`
	MergedIntoName  string `json:"mergedIntoName"`
	MergedAt        int64  `json:"mergedAt"`
	RecordCount     int64  `json:"recordCount"`
	InCent          int64  `json:"inCent"`
	OutCent         int64  `json:"outCent"`
	PendingCount    int64  `json:"pendingCount"`
	CreatedAt       int64  `json:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type ItemPart struct {
	ItemKind      string `json:"itemKind"`
	ItemKindLabel string `json:"itemKindLabel"`
	Direction     string `json:"direction"`
	Cent          int64  `json:"cent"`
	RecordCount   int64  `json:"recordCount"`
}

type Detail struct {
	Resp
	ByItem []ItemPart `json:"byItem"`
}

type CreateParam struct {
	Name           string `json:"name"`
	Category       string `json:"category"`
	DefaultEventID uint64 `json:"defaultEventId"`
	Description    string `json:"description"`
}

type UpdateParam struct {
	Name           string `json:"name"`
	Category       string `json:"category"`
	DefaultEventID *uint64 `json:"defaultEventId"`
	Description    string `json:"description"`
}

type MergeParam struct {
	SourceIDs []uint64 `json:"sourceIds"`
	TargetID  uint64   `json:"targetId"`
}

type MergePreview struct {
	Target        Resp        `json:"target"`
	Sources       []Resp      `json:"sources"`
	RecordCount   int64       `json:"recordCount"`
	CrossCategory bool        `json:"crossCategory"`
	Categories    []string    `json:"categories"`
	ItemCounts    []ItemCount `json:"itemCounts"`
}

type ItemCount struct {
	ItemKind      string `json:"itemKind"`
	ItemKindLabel string `json:"itemKindLabel"`
	RecordCount   int64  `json:"recordCount"`
}

type MergeSourceSnapshot struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	RecordCount int64  `json:"recordCount"`
}

type MergeLog struct {
	ID            uint64                 `json:"id"`
	TargetBookID  uint64                 `json:"targetBookId"`
	TargetName    string                 `json:"targetName"`
	Sources       []MergeSourceSnapshot  `json:"sources"`
	RecordCount   int64                  `json:"recordCount"`
	CrossCategory bool                   `json:"crossCategory"`
	CreatedAt     int64                  `json:"createdAt"`
}

// Agg 是礼簿维度的汇总，由一条 GROUP BY 查询批量取得。
type Agg struct {
	RecordCount  int64 `json:"recordCount"`
	InCent       int64 `json:"inCent"`
	OutCent      int64 `json:"outCent"`
	PendingCount int64 `json:"pendingCount"`
}

type PaperPage struct {
	Page         string `json:"page"`
	DeclaredCent int64  `json:"declaredCent"`
	RecordCent   int64  `json:"recordCent"`
	RecordCount  int64  `json:"recordCount"`
	HasDeclared  bool   `json:"hasDeclared"`
	Consistent   bool   `json:"consistent"`
	DiffCent     int64  `json:"diffCent"`
}

type PaperPageDetail struct {
	PaperPage
	Records []record.Resp `json:"records"`
}

type MergeLogRow struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	OwnerID       uint64    `gorm:"column:owner_id" json:"ownerId"`
	TargetBookID  uint64    `gorm:"column:target_book_id" json:"targetBookId"`
	SourceBooks   string    `gorm:"column:source_books" json:"sourceBooks"`
	RecordCount   int64     `gorm:"column:record_count" json:"recordCount"`
	CrossCategory bool      `gorm:"column:cross_category" json:"crossCategory"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"-"`
}

func (*MergeLogRow) TableName() string { return "book_merge_log" }
