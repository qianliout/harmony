// Package dict 定义本子类别与记录名目字典的模型、入参与出参。
package dict

import "time"

type Item struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	OwnerID   uint64    `gorm:"column:owner_id" json:"ownerId"`
	DictType  string    `gorm:"column:dict_type" json:"type"`
	Code      string    `gorm:"column:code" json:"code"`
	Label     string    `gorm:"column:label" json:"label"`
	Color     string    `gorm:"column:color" json:"color"`
	Sort      int       `gorm:"column:sort" json:"sort"`
	Builtin   bool      `gorm:"column:builtin" json:"builtin"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

func (*Item) TableName() string { return "dict_item" }

type Resp struct {
	ID      uint64 `json:"id"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Label   string `json:"label"`
	Color   string `json:"color"`
	Sort    int    `json:"sort"`
	Builtin bool   `json:"builtin"`
}

func (m *Item) Serialize() Resp {
	return Resp{
		ID:      m.ID,
		Type:    m.DictType,
		Code:    m.Code,
		Label:   m.Label,
		Color:   m.Color,
		Sort:    m.Sort,
		Builtin: m.Builtin,
	}
}

type CreateParam struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Color string `json:"color"`
}

type UpdateParam struct {
	Label string `json:"label"`
	Color string `json:"color"`
	Sort  *int   `json:"sort"`
}
