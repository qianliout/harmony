// Package setting 定义跨端同步的偏好设置。
package setting

import (
	"time"

	"harmony/infra/consts"
)

type Row struct {
	OwnerID   uint64    `gorm:"column:owner_id;primaryKey" json:"-"`
	K         string    `gorm:"column:k;primaryKey" json:"-"`
	V         string    `gorm:"column:v" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

func (*Row) TableName() string { return "user_setting" }

type Settings struct {
	ShowOwing    bool    `json:"showOwing"`
	LastBookID   uint64  `json:"lastBookId"`
	LastEventID  uint64  `json:"lastEventId"`
	QuickAmounts []int64 `json:"quickAmounts"`
}

func Default() Settings {
	return Settings{
		ShowOwing:    true,
		QuickAmounts: append([]int64(nil), consts.DefaultQuickAmounts...),
	}
}

type UpdateParam struct {
	ShowOwing    *bool    `json:"showOwing"`
	LastBookID   *uint64  `json:"lastBookId"`
	LastEventID  *uint64  `json:"lastEventId"`
	QuickAmounts *[]int64 `json:"quickAmounts"`
}
