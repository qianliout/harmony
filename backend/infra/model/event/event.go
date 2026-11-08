// Package event 定义事项模型、入参与出参。
package event

import (
	"strings"
	"time"

	"harmony/infra/consts"
)

type Event struct {
	ID             uint64     `gorm:"primaryKey" json:"id"`
	OwnerID        uint64     `gorm:"column:owner_id" json:"ownerId"`
	Name           string     `gorm:"column:name" json:"name"`
	Type           string     `gorm:"column:type" json:"type"`
	EventDate      *time.Time `gorm:"column:event_date;type:date" json:"-"`
	Location       string     `gorm:"column:location" json:"location"`
	Host           string     `gorm:"column:host" json:"host"`
	HostPersonID   *uint64    `gorm:"column:host_person_id" json:"hostPersonId"`
	DatePrecision  string     `gorm:"column:date_precision" json:"datePrecision"`
	Placeholder    bool       `gorm:"column:placeholder" json:"placeholder"`
	Remark         string     `gorm:"column:remark" json:"remark"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"-"`
}

func (*Event) TableName() string { return "event" }

type Resp struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	TypeLabel      string `json:"typeLabel"`
	EventDate      string `json:"eventDate"`
	DatePrecision  string `json:"datePrecision"`
	Location       string `json:"location"`
	Host           string `json:"host"`
	HostLabel      string `json:"hostLabel"`
	HostPersonID   uint64 `json:"hostPersonId"`
	HostPersonName string `json:"hostPersonName"`
	Placeholder    bool   `json:"placeholder"`
	Remark         string `json:"remark"`
	RecordCount    int64  `json:"recordCount"`
	InCent         int64  `json:"inCent"`
	OutCent        int64  `json:"outCent"`
	PersonCount    int64  `json:"personCount"`
	RepaidCount    int64  `json:"repaidCount"`
	PendingCount   int64  `json:"pendingCount"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
}

// Summary 是事项维度的汇总。
type Summary struct {
	InCent       int64 `json:"inCent"`
	OutCent      int64 `json:"outCent"`
	PersonCount  int64 `json:"personCount"`
	RepaidCount  int64 `json:"repaidCount"`
	PendingCount int64 `json:"pendingCount"`
	RecordCount  int64 `json:"recordCount"`
}

type Detail struct {
	Resp
	Summary Summary `json:"summary"`
}

// InferTypeByName 按事项名关键字推断类型，用于纸本标注补录时自动新建事项。
func InferTypeByName(name string) string {
	switch {
	case containsAny(name, "结婚", "嫁", "娶", "婚宴", "喜酒"):
		return consts.EventTypeWedding
	case containsAny(name, "去世", "丧", "白事", "过世"):
		return consts.EventTypeFuneral
	case containsAny(name, "满月", "办儿", "办女", "百日"):
		return consts.EventTypeFullMoon
	case containsAny(name, "进屋", "乔迁", "新房", "搬家"):
		return consts.EventTypeHousewarming
	case containsAny(name, "升学", "考上", "大学", "高考"):
		return consts.EventTypeSchool
	case containsAny(name, "过年", "春节", "新年"):
		return consts.EventTypeNewYear
	default:
		return consts.EventTypeOther
	}
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func LabelOfType(eventType string) string {
	if label, ok := consts.EventTypeLabels[eventType]; ok {
		return label
	}
	return consts.EventTypeLabels[consts.EventTypeOther]
}

func LabelOfHost(host string) string {
	if host == consts.EventHostSelf {
		return "自家办"
	}
	return "别人家办"
}

type CreateParam struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	EventDate     string `json:"eventDate"`
	DatePrecision string `json:"datePrecision"`
	Location      string `json:"location"`
	Host          string `json:"host"`
	HostPersonID  uint64 `json:"hostPersonId"`
	Remark        string `json:"remark"`
}

type UpdateParam struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	EventDate     string `json:"eventDate"`
	DatePrecision string `json:"datePrecision"`
	Location      string `json:"location"`
	Host          string `json:"host"`
	HostPersonID  uint64 `json:"hostPersonId"`
	Remark        string `json:"remark"`
}

type Query struct {
	Type             string `json:"type"`
	Host             string `json:"host"`
	From             string `json:"from"`
	To               string `json:"to"`
	Keyword          string `json:"keyword"`
	IncludePlaceholder bool `json:"includePlaceholder"`
}
