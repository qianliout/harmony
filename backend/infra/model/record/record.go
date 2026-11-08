// Package record 定义人情记录、还礼关联、纸本标注与按页核对的模型与出入参。
package record

import (
	"time"

	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/utils"
)

type Record struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	OwnerID          uint64    `gorm:"column:owner_id" json:"ownerId"`
	PersonID         uint64    `gorm:"column:person_id" json:"personId"`
	EventID          uint64    `gorm:"column:event_id" json:"eventId"`
	BookID           uint64    `gorm:"column:book_id" json:"bookId"`
	OriginBookID     uint64    `gorm:"column:origin_book_id" json:"originBookId"`
	ItemKind         string    `gorm:"column:item_kind" json:"itemKind"`
	Direction        string    `gorm:"column:direction" json:"direction"`
	AmountCent       int64     `gorm:"column:amount_cent" json:"amountCent"`
	GiftName         string    `gorm:"column:gift_name" json:"giftName"`
	GiftValueCent    int64     `gorm:"column:gift_value_cent" json:"giftValueCent"`
	TotalCent        int64     `gorm:"column:total_cent;->" json:"totalCent"`
	OccurredOn       utils.Date `gorm:"column:occurred_on" json:"-"`
	DatePrecision    string    `gorm:"column:date_precision" json:"datePrecision"`
	Status           string    `gorm:"column:status" json:"status"`
	Remark           string    `gorm:"column:remark" json:"remark"`
	PaperNote        string    `gorm:"column:paper_note" json:"paperNote"`
	PaperPage        string    `gorm:"column:paper_page" json:"paperPage"`
	PaperSeq         int       `gorm:"column:paper_seq" json:"paperSeq"`
	Source           string    `gorm:"column:source" json:"source"`
	RecognizeTaskID  *uint64   `gorm:"column:recognize_task_id" json:"recognizeTaskId"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"-"`
}

func (*Record) TableName() string { return "record" }

type Repay struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	OwnerID        uint64    `gorm:"column:owner_id" json:"ownerId"`
	RepayRecordID  uint64    `gorm:"column:repay_record_id" json:"repayRecordId"`
	OriginRecordID uint64    `gorm:"column:origin_record_id" json:"originRecordId"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"-"`
}

func (*Repay) TableName() string { return "record_repay" }

type PaperPageCheck struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	OwnerID      uint64    `gorm:"column:owner_id" json:"ownerId"`
	BookID       uint64    `gorm:"column:book_id" json:"bookId"`
	PaperPage    string    `gorm:"column:paper_page" json:"paperPage"`
	DeclaredCent int64     `gorm:"column:declared_cent" json:"declaredCent"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"-"`
}

func (*PaperPageCheck) TableName() string { return "paper_page_check" }

// Resp 是记录的读出口；关联名称由 service 批量补齐。
type Resp struct {
	ID             uint64  `json:"id"`
	PersonID       uint64  `json:"personId"`
	PersonName     string  `json:"personName"`
	PersonRelation string  `json:"personRelation"`
	EventID        uint64  `json:"eventId"`
	EventName      string  `json:"eventName"`
	EventType      string  `json:"eventType"`
	BookID         uint64  `json:"bookId"`
	BookName       string  `json:"bookName"`
	OriginBookID   uint64  `json:"originBookId"`
	OriginBookName string  `json:"originBookName"`
	ItemKind       string  `json:"itemKind"`
	ItemKindLabel  string  `json:"itemKindLabel"`
	Direction      string  `json:"direction"`
	AmountCent     int64   `json:"amountCent"`
	GiftName       string  `json:"giftName"`
	GiftValueCent  int64   `json:"giftValueCent"`
	TotalCent      int64   `json:"totalCent"`
	OccurredOn     string  `json:"occurredOn"`
	DatePrecision  string  `json:"datePrecision"`
	Status         string  `json:"status"`
	StatusLabel    string  `json:"statusLabel"`
	IsRepay        bool    `json:"isRepay"`
	Remark         string  `json:"remark"`
	PaperNote      string  `json:"paperNote"`
	PaperPage      string  `json:"paperPage"`
	PaperSeq       int     `json:"paperSeq"`
	Source         string  `json:"source"`
	RecognizeTaskID uint64 `json:"recognizeTaskId"`
	RepayCount     int64   `json:"repayCount"`
	CreatedAt      int64   `json:"createdAt"`
	UpdatedAt      int64   `json:"updatedAt"`
	Confidence     float64 `json:"confidence,omitempty"`
}

type Detail struct {
	Resp
	RepayRecords  []Resp `json:"repayRecords"`
	OriginRecords []Resp `json:"originRecords"`
}

// Intent 是搜索意图解析结果，所有列表接口共用。
type Intent struct {
	Kind         string `json:"kind"`
	Text         string `json:"text"`
	AmountCent   int64  `json:"amountCent"`
	AmountMinCent int64 `json:"amountMinCent"`
	AmountMaxCent int64 `json:"amountMaxCent"`
	Year         int64  `json:"year"`
	PaperPage    string `json:"paperPage"`
}

type Query struct {
	BookID       uint64 `json:"bookId"`
	OriginBookID uint64 `json:"originBookId"`
	BookCategory string `json:"bookCategory"`
	EventType    string `json:"eventType"`
	PersonID  uint64 `json:"personId"`
	EventID   uint64 `json:"eventId"`
	ItemKind  string `json:"itemKind"`
	Direction string `json:"direction"`
	Status    string `json:"status"`
	From      string `json:"from"`
	To        string `json:"to"`
	Keyword   string `json:"keyword"`
	Intent    Intent `json:"-"`
}

type CreateParam struct {
	PersonID        uint64 `json:"personId"`
	PersonName      string `json:"personName"`
	EventID         uint64 `json:"eventId"`
	BookID          uint64 `json:"bookId"`
	ItemKind        string `json:"itemKind"`
	Direction       string `json:"direction"`
	AmountCent      int64  `json:"amountCent"`
	GiftName        string `json:"giftName"`
	GiftValueCent   int64  `json:"giftValueCent"`
	OccurredOn      string `json:"occurredOn"`
	DatePrecision   string `json:"datePrecision"`
	Status          string `json:"status"`
	Remark          string `json:"remark"`
	PaperNote       string `json:"paperNote"`
	PaperPage       string `json:"paperPage"`
	PaperSeq        int    `json:"paperSeq"`
	Source          string `json:"source"`
	RecognizeTaskID *uint64 `json:"-"`
	Confirm         bool   `json:"confirm"`
}

type UpdateParam struct {
	PersonID      uint64 `json:"personId"`
	EventID       uint64 `json:"eventId"`
	BookID        uint64 `json:"bookId"`
	ItemKind      string `json:"itemKind"`
	Direction     string `json:"direction"`
	AmountCent    int64  `json:"amountCent"`
	GiftName      string `json:"giftName"`
	GiftValueCent int64  `json:"giftValueCent"`
	OccurredOn    string `json:"occurredOn"`
	DatePrecision string `json:"datePrecision"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	PaperNote     string `json:"paperNote"`
	PaperPage     string `json:"paperPage"`
	PaperSeq      int    `json:"paperSeq"`
	Confirm       bool   `json:"confirm"`
}

type RepayItem struct {
	BookID       uint64 `json:"bookId"`
	ItemKind     string `json:"itemKind"`
	AmountCent   int64  `json:"amountCent"`
	GiftName     string `json:"giftName"`
	GiftValueCent int64 `json:"giftValueCent"`
}

type EventParam struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Date          string `json:"date"`
	DatePrecision string `json:"datePrecision"`
}

type RepayParam struct {
	OriginIDs     []uint64    `json:"originIds"`
	MarkRepaid    *bool       `json:"markRepaid"`
	Event         *EventParam `json:"event"`
	OccurredOn    string      `json:"occurredOn"`
	DatePrecision string      `json:"datePrecision"`
	PaperNote     string      `json:"paperNote"`
	Items         []RepayItem `json:"items"`
	Remark        string      `json:"remark"`
}

type MarkRepaidParam struct {
	PaperNote string `json:"paperNote"`
}

type RepayResult struct {
	Records    []Resp `json:"records"`
	RepayCount int    `json:"repayCount"`
}

type UnlinkParam struct {
	OriginID uint64 `json:"originId"`
}

// PaperDraft 是纸本标注解析出的单次还礼草稿，结构与还礼请求体一致。
type PaperDraft struct {
	Event         *EventParam `json:"event"`
	OccurredOn    string      `json:"occurredOn"`
	DatePrecision string      `json:"datePrecision"`
	Items         []RepayItem `json:"items"`
	Raw           string      `json:"raw"`
	Confidence    string      `json:"confidence"`
}

type PaperParseResult struct {
	OnlyMark bool         `json:"onlyMark"`
	Repays   []PaperDraft `json:"repays"`
}

type PaperNoteParseParam struct {
	Note     string `json:"note"`
	PersonID uint64 `json:"personId"`
}

type PageAgg struct {
	PaperPage   string `gorm:"column:paper_page" json:"paperPage"`
	RecordCent  int64  `gorm:"column:record_cent" json:"recordCent"`
	RecordCount int64  `gorm:"column:record_count" json:"recordCount"`
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
	Records []Resp `json:"records"`
}

// StatusLabel 按方向和是否为回礼生成展示文字。任何页面都不显示不带方向的“待还”。
func StatusLabel(status, direction string, isRepay bool) string {
	if isRepay {
		return "回礼"
	}
	if direction == consts.DirectionOut {
		switch status {
		case consts.RecordStatusRepaid:
			return "对方已回"
		case consts.RecordStatusNone:
			return "不用回"
		default:
			return "对方未回"
		}
	}
	switch status {
	case consts.RecordStatusRepaid:
		return "已回礼"
	case consts.RecordStatusNone:
		return "不用回"
	default:
		return "未回礼"
	}
}

func PageFilter() *modelfilter.Filter { return &modelfilter.Filter{} }
