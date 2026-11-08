// Package person 定义亲友模型、入参与出参。
package person

import (
	"time"

	"harmony/infra/consts"
)

type Person struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	OwnerID      uint64    `gorm:"column:owner_id" json:"ownerId"`
	Name         string    `gorm:"column:name" json:"name"`
	NamePinyin   string    `gorm:"column:name_pinyin" json:"-"`
	NameInitials string    `gorm:"column:name_initials" json:"-"`
	Relation     string    `gorm:"column:relation" json:"relation"`
	Phone        string    `gorm:"column:phone" json:"phone"`
	Remark       string    `gorm:"column:remark" json:"remark"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"-"`
}

func (*Person) TableName() string { return "person" }

// Summary 是人维度的统计口径，所有金额为分。
type Summary struct {
	InCent       int64  `json:"inCent"`
	OutCent      int64  `json:"outCent"`
	NetCent      int64  `json:"netCent"`
	PendingCount int64  `json:"pendingCount"`
	PendingCent  int64  `json:"pendingCent"`
	RecordCount  int64  `json:"recordCount"`
	LastDate     string `json:"lastDate"`
}

type Brief struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Relation      string `json:"relation"`
	RelationLabel string `json:"relationLabel"`
	Phone         string `json:"phone"`
	Remark        string `json:"remark"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
	InCent        int64  `json:"inCent"`
	OutCent       int64  `json:"outCent"`
	PendingCount  int64  `json:"pendingCount"`
	LastDate      string `json:"lastDate"`
}

type Detail struct {
	ID            uint64  `json:"id"`
	Name          string  `json:"name"`
	Relation      string  `json:"relation"`
	RelationLabel string  `json:"relationLabel"`
	Phone         string  `json:"phone"`
	Remark        string  `json:"remark"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	Summary       Summary `json:"summary"`
}

// Ref 是聚合视图里的精简人信息。
type Ref struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Relation      string `json:"relation"`
	RelationLabel string `json:"relationLabel"`
}

func (p *Person) Ref() Ref {
	return Ref{ID: p.ID, Name: p.Name, Relation: p.Relation, RelationLabel: LabelOf(p.Relation)}
}

func LabelOf(relation string) string {
	if label, ok := consts.RelationLabels[relation]; ok {
		return label
	}
	return consts.RelationLabels[consts.RelationOther]
}

type CreateParam struct {
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Phone    string `json:"phone"`
	Remark   string `json:"remark"`
	Confirm  bool   `json:"confirm"`
}

type UpdateParam struct {
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Phone    string `json:"phone"`
	Remark   string `json:"remark"`
	Confirm  bool   `json:"confirm"`
}

type Query struct {
	Keyword  string `json:"keyword"`
	Relation string `json:"relation"`
}
