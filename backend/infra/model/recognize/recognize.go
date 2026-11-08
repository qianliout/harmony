// Package recognize 定义识别任务、识别项与批量写入入参。
package recognize

import (
	"time"

	"harmony/infra/model/record"
)

type Task struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	OwnerID      uint64     `gorm:"column:owner_id" json:"ownerId"`
	ImagePath    string     `gorm:"column:image_path" json:"imagePath"`
	ImageSHA256  string     `gorm:"column:image_sha256" json:"imageSha256"`
	ImageSize    int        `gorm:"column:image_size" json:"imageSize"`
	Status       string     `gorm:"column:status" json:"status"`
	Provider     string     `gorm:"column:provider" json:"provider"`
	Model        string     `gorm:"column:model" json:"model"`
	RawResponse  string     `gorm:"column:raw_response" json:"-"`
	Items        string     `gorm:"column:items;type:json" json:"-"`
	ErrorMessage string     `gorm:"column:error_message" json:"errorMessage"`
	Committed    bool       `gorm:"column:committed" json:"committed"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"-"`
	FinishedAt   *time.Time `gorm:"column:finished_at" json:"-"`
}

func (*Task) TableName() string { return "recognize_task" }

// Match 表示识别出的姓名与已有亲友的匹配情况。
type Match struct {
	Status     string        `json:"status"` // single | none | multiple
	PersonID   uint64        `json:"personId"`
	Candidates []PersonMatch `json:"candidates"`
}

type PersonMatch struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	RelationLabel string `json:"relationLabel"`
	LastDate      string `json:"lastDate"`
}

type Item struct {
	Seq           int                 `json:"seq"`
	Name          string              `json:"name"`
	AmountUpper   string              `json:"amountUpper"`
	AmountLower   *float64            `json:"amountLower"`
	Amount        float64             `json:"amount"`
	GiftName      string              `json:"giftName"`
	ItemKind      string              `json:"itemKind"`
	Direction     string              `json:"direction"`
	Date          string              `json:"date"`
	Note          string              `json:"note"`
	Raw           string              `json:"raw"`
	Confidence    float64             `json:"confidence"`
	AmountMismatch bool               `json:"amountMismatch"`
	Match         *Match              `json:"match,omitempty"`
	RepayDrafts   []record.PaperDraft `json:"repayDrafts,omitempty"`
	DuplicateOf   *uint64             `json:"duplicateOf,omitempty"`
}

type Resp struct {
	ID           uint64 `json:"id"`
	Status       string `json:"status"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	ImagePath    string `json:"imagePath"`
	ImageURL     string `json:"imageUrl"`
	ImageSHA256  string `json:"imageSha256"`
	ImageSize    int    `json:"imageSize"`
	ErrorMessage string `json:"errorMessage"`
	Committed    bool   `json:"committed"`
	Page         string `json:"page"`
	PageTotalCent int64 `json:"pageTotalCent"`
	SumCent      int64 `json:"sumCent"`
	Items        []Item `json:"items"`
	CreatedAt    int64 `json:"createdAt"`
	FinishedAt   int64 `json:"finishedAt"`
}

// CommitItem 是核对页提交的一行：与新建记录字段一致，附带要一并补录的还礼草稿。
type CommitItem struct {
	PersonID      uint64              `json:"personId"`
	PersonName    string              `json:"personName"`
	EventID       uint64              `json:"eventId"`
	BookID        uint64              `json:"bookId"`
	ItemKind      string              `json:"itemKind"`
	Direction     string              `json:"direction"`
	AmountCent    int64               `json:"amountCent"`
	GiftName      string              `json:"giftName"`
	GiftValueCent int64               `json:"giftValueCent"`
	OccurredOn    string              `json:"occurredOn"`
	DatePrecision string              `json:"datePrecision"`
	Status        string              `json:"status"`
	Remark        string              `json:"remark"`
	PaperNote     string              `json:"paperNote"`
	PaperPage     string              `json:"paperPage"`
	PaperSeq      int                 `json:"paperSeq"`
	Repays        []record.RepayParam `json:"repays"`
}

type CommitParam struct {
	Items     []CommitItem `json:"items"`
	Page      string       `json:"page"`
	PageTotalCent *int64   `json:"pageTotalCent"`
	BookID    uint64       `json:"bookId"`
}

type CommitResult struct {
	RecordIDs    []uint64 `json:"recordIds"`
	Count        int      `json:"count"`
	NewPersons   int      `json:"newPersons"`
	RepayCount   int      `json:"repayCount"`
	SumCent      int64    `json:"sumCent"`
}

// Provider 是模型返回的原始结构化结果。
type ProviderResult struct {
	Model string
	Raw   string
	Items []ProviderItem
	Page  string
	PageTotal *float64
}

type ProviderItem struct {
	Seq         int      `json:"seq"`
	Name        string   `json:"name"`
	AmountUpper string   `json:"amountUpper"`
	AmountLower *float64 `json:"amountLower"`
	Amount      float64  `json:"amount"`
	GiftName    string   `json:"giftName"`
	ItemKind    string   `json:"itemKind"`
	Direction   string   `json:"direction"`
	Date        string   `json:"date"`
	Note        string   `json:"note"`
	Raw         string   `json:"raw"`
	Confidence  float64  `json:"confidence"`
}
