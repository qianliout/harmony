// Package record 定义人情记录的模型、入参、出参与统计结构。
package record

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm/clause"

	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/utils"
)

// UniqueID: not generated because 同一人同一天可以有多笔相同金额的往来，没有稳定业务键。
type Record struct {
	ID int64 `gorm:"primaryKey;column:id" json:"id"`

	OwnerID    int64  `gorm:"column:owner_id" json:"ownerId"`
	PersonID   int64  `gorm:"column:person_id" json:"personId"`
	BookID     int64  `gorm:"column:book_id" json:"bookId"`
	Direction  string `gorm:"column:direction" json:"direction"`
	AmountCent int64  `gorm:"column:amount_cent" json:"amountCent"`
	Matter     string `gorm:"column:matter" json:"matter"`
	Note       string `gorm:"column:note" json:"note"`
	OccurredOn string `gorm:"column:occurred_on" json:"occurredOn"`

	CreatedAt int64 `gorm:"autoCreateTime:milli;column:created_at" json:"createdAt"`
	UpdatedAt int64 `gorm:"autoUpdateTime:milli;column:updated_at" json:"updatedAt"`
}

func (vi *Record) TableName() string { return "record" }

// Serialize 落实「收挂礼簿、送记事情」：送礼清空礼簿，收礼清空事情。
func (vi *Record) Serialize() *Record {
	if vi == nil {
		return &Record{}
	}
	vi.Direction = strings.TrimSpace(vi.Direction)
	vi.Matter = strings.TrimSpace(vi.Matter)
	vi.Note = strings.TrimSpace(vi.Note)
	vi.OccurredOn = strings.TrimSpace(vi.OccurredOn)
	switch vi.Direction {
	case consts.DirectionOut:
		vi.BookID = 0
	case consts.DirectionIn:
		vi.Matter = ""
	}
	return vi
}

func (vi *Record) Deserialize() *Record {
	if vi == nil {
		return &Record{}
	}
	vi.CreatedAt = utils.MilliTimestamp(vi.CreatedAt)
	vi.UpdatedAt = utils.MilliTimestamp(vi.UpdatedAt)
	return vi
}

func (vi *Record) Check() error {
	if vi == nil {
		return fmt.Errorf("记录数据为空")
	}
	if vi.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if vi.PersonID <= 0 {
		return fmt.Errorf("请选择亲友")
	}
	if !consts.ValidDirection(vi.Direction) {
		return fmt.Errorf("方向只能是收或送")
	}
	inWithoutBook := vi.Direction == consts.DirectionIn && vi.BookID <= 0
	if inWithoutBook {
		return fmt.Errorf("收礼需要选择礼簿")
	}
	if vi.AmountCent < 0 {
		return fmt.Errorf("金额不能为负数")
	}
	if _, err := utils.ParseDate(vi.OccurredOn); err != nil {
		return fmt.Errorf("日期格式应为 YYYY-MM-DD")
	}
	if utf8.RuneCountInString(vi.Matter) > consts.RecordMatterMaxLen {
		return fmt.Errorf("事情不能超过 %d 个字", consts.RecordMatterMaxLen)
	}
	if utf8.RuneCountInString(vi.Note) > consts.RecordNoteMaxLen {
		return fmt.Errorf("备注不能超过 %d 个字", consts.RecordNoteMaxLen)
	}
	return nil
}

// ToUpdater 供完整更新使用：礼簿与事情随方向派生，必须整体写入。
func (vi *Record) ToUpdater() map[string]any {
	return map[string]any{
		"person_id":   vi.PersonID,
		"book_id":     vi.BookID,
		"direction":   vi.Direction,
		"amount_cent": vi.AmountCent,
		"matter":      vi.Matter,
		"note":        vi.Note,
		"occurred_on": vi.OccurredOn,
		"updated_at":  time.Now().UTC().UnixMilli(),
	}
}

// CreateRecordParam：PersonID 为正时直接使用；否则按姓名与备注匹配，找不到就新建亲友。
type CreateRecordParam struct {
	OwnerID      int64  `json:"-"`
	PersonID     int64  `json:"personId"`
	PersonName   string `json:"personName"`
	PersonRemark string `json:"personRemark"`
	BookID       int64  `json:"bookId"`
	Direction    string `json:"direction"`
	AmountCent   int64  `json:"amountCent"`
	Matter       string `json:"matter"`
	Note         string `json:"note"`
	OccurredOn   string `json:"occurredOn"`
}

func (p *CreateRecordParam) Serialize() *CreateRecordParam {
	if p == nil {
		return &CreateRecordParam{}
	}
	p.PersonName = strings.TrimSpace(p.PersonName)
	p.PersonRemark = strings.TrimSpace(p.PersonRemark)
	p.Direction = strings.TrimSpace(p.Direction)
	p.Matter = strings.TrimSpace(p.Matter)
	p.Note = strings.TrimSpace(p.Note)
	p.OccurredOn = strings.TrimSpace(p.OccurredOn)
	if p.Direction == "" {
		p.Direction = consts.DirectionIn
	}
	if p.OccurredOn == "" {
		p.OccurredOn = utils.DateString(time.Now())
	}
	if p.Direction == consts.DirectionOut {
		p.BookID = 0
	}
	return p
}

func (p *CreateRecordParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.PersonID <= 0 && p.PersonName == "" {
		return fmt.Errorf("请选择或填写亲友")
	}
	if !consts.ValidDirection(p.Direction) {
		return fmt.Errorf("方向只能是收或送")
	}
	inWithoutBook := p.Direction == consts.DirectionIn && p.BookID <= 0
	if inWithoutBook {
		return fmt.Errorf("收礼需要选择礼簿")
	}
	if p.AmountCent < 0 {
		return fmt.Errorf("金额不能为负数")
	}
	if _, err := utils.ParseDate(p.OccurredOn); err != nil {
		return fmt.Errorf("日期格式应为 YYYY-MM-DD")
	}
	return nil
}

// UpdateRecordParam 是完整更新：id 来自 query，其余字段来自 body。
type UpdateRecordParam struct {
	ID         int64  `json:"-"`
	OwnerID    int64  `json:"-"`
	PersonID   int64  `json:"personId"`
	BookID     int64  `json:"bookId"`
	Direction  string `json:"direction"`
	AmountCent int64  `json:"amountCent"`
	Matter     string `json:"matter"`
	Note       string `json:"note"`
	OccurredOn string `json:"occurredOn"`
}

func (p *UpdateRecordParam) Serialize() *UpdateRecordParam {
	if p == nil {
		return &UpdateRecordParam{}
	}
	p.Direction = strings.TrimSpace(p.Direction)
	p.Matter = strings.TrimSpace(p.Matter)
	p.Note = strings.TrimSpace(p.Note)
	p.OccurredOn = strings.TrimSpace(p.OccurredOn)
	if p.Direction == consts.DirectionOut {
		p.BookID = 0
	}
	return p
}

func (p *UpdateRecordParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少记录 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.PersonID <= 0 {
		return fmt.Errorf("请选择亲友")
	}
	if !consts.ValidDirection(p.Direction) {
		return fmt.Errorf("方向只能是收或送")
	}
	inWithoutBook := p.Direction == consts.DirectionIn && p.BookID <= 0
	if inWithoutBook {
		return fmt.Errorf("收礼需要选择礼簿")
	}
	if p.AmountCent < 0 {
		return fmt.Errorf("金额不能为负数")
	}
	if _, err := utils.ParseDate(p.OccurredOn); err != nil {
		return fmt.Errorf("日期格式应为 YYYY-MM-DD")
	}
	return nil
}

// GetRecordParam 用于详情与删除。
type GetRecordParam struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"-"`
}

func (p *GetRecordParam) Serialize() *GetRecordParam {
	if p == nil {
		return &GetRecordParam{}
	}
	return p
}

func (p *GetRecordParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少记录 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	return nil
}

// SearchRecordParam 统一承载总账、礼簿详情、亲友详情三种视图的筛选。
// From/To 为闭区间日期；Keyword 匹配备注、事情与亲友姓名、备注、拼音。
type SearchRecordParam struct {
	OwnerID   int64               `json:"-"`
	RecordIDs []int64             `json:"recordIds"`
	PersonID  int64               `json:"personId"`
	BookID    int64               `json:"bookId"`
	Direction string              `json:"direction"`
	From      string              `json:"from"`
	To        string              `json:"to"`
	Keyword   string              `json:"keyword"`
	Filter    *modelfilter.Filter `json:"filter"`
}

func (p *SearchRecordParam) Serialize() *SearchRecordParam {
	if p == nil {
		return &SearchRecordParam{Filter: &modelfilter.Filter{}}
	}
	p.Direction = strings.TrimSpace(p.Direction)
	p.From = strings.TrimSpace(p.From)
	p.To = strings.TrimSpace(p.To)
	p.Keyword = strings.TrimSpace(p.Keyword)
	if p.RecordIDs == nil {
		p.RecordIDs = []int64{}
	}
	if p.Filter == nil {
		p.Filter = &modelfilter.Filter{}
	}
	noSort := p.Filter.SortField == "" && len(p.Filter.OrderByColumns) == 0
	if noSort {
		p.Filter.OrderByColumns = []clause.OrderByColumn{
			{Column: clause.Column{Name: "occurred_on"}, Desc: true},
			{Column: clause.Column{Name: "id"}, Desc: true},
		}
	}
	return p
}

func (p *SearchRecordParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	invalidDirection := p.Direction != "" && !consts.ValidDirection(p.Direction)
	if invalidDirection {
		return fmt.Errorf("方向只能是收或送")
	}
	if err := checkDateBound(p.From, p.To); err != nil {
		return err
	}
	if !consts.RecordSortFields[p.Filter.SortField] {
		return fmt.Errorf("不支持按 %s 排序", p.Filter.SortField)
	}
	return nil
}

// StatRecordParam 与 SearchRecordParam 同一套筛选口径，另按 GroupBy 分组；PersonIDs/BookIDs 用于批量取合计。
type StatRecordParam struct {
	OwnerID   int64   `json:"-"`
	PersonID  int64   `json:"personId"`
	BookID    int64   `json:"bookId"`
	PersonIDs []int64 `json:"personIds"`
	BookIDs   []int64 `json:"bookIds"`
	Direction string  `json:"direction"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Keyword   string  `json:"keyword"`
	GroupBy   string  `json:"groupBy"`
}

func (p *StatRecordParam) Serialize() *StatRecordParam {
	if p == nil {
		return &StatRecordParam{}
	}
	p.Direction = strings.TrimSpace(p.Direction)
	p.From = strings.TrimSpace(p.From)
	p.To = strings.TrimSpace(p.To)
	p.Keyword = strings.TrimSpace(p.Keyword)
	if p.PersonIDs == nil {
		p.PersonIDs = []int64{}
	}
	if p.BookIDs == nil {
		p.BookIDs = []int64{}
	}
	return p
}

func (p *StatRecordParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	invalidDirection := p.Direction != "" && !consts.ValidDirection(p.Direction)
	if invalidDirection {
		return fmt.Errorf("方向只能是收或送")
	}
	if err := checkDateBound(p.From, p.To); err != nil {
		return err
	}
	switch p.GroupBy {
	case consts.RecordGroupByNone, consts.RecordGroupByPerson, consts.RecordGroupByBook:
		return nil
	}
	return fmt.Errorf("unsupported groupBy %q", p.GroupBy)
}

// checkDateBound 被 SearchRecordParam 与 StatRecordParam 共用，两者筛选口径必须一致。
func checkDateBound(from, to string) error {
	if from != "" {
		if _, err := utils.ParseDate(from); err != nil {
			return fmt.Errorf("开始日期格式应为 YYYY-MM-DD")
		}
	}
	if to != "" {
		if _, err := utils.ParseDate(to); err != nil {
			return fmt.Errorf("结束日期格式应为 YYYY-MM-DD")
		}
	}
	return nil
}

// RecordView 是记录的读出口；亲友与礼簿名称由 service 批量补齐。
type RecordView struct {
	ID                int64  `json:"id"`
	PersonID          int64  `json:"personId"`
	PersonName        string `json:"personName"`
	PersonRemark      string `json:"personRemark"`
	PersonDisplayName string `json:"personDisplayName"`
	BookID            int64  `json:"bookId"`
	BookName          string `json:"bookName"`
	Direction         string `json:"direction"`
	AmountCent        int64  `json:"amountCent"`
	Matter            string `json:"matter"`
	Note              string `json:"note"`
	OccurredOn        string `json:"occurredOn"`
	CreatedAt         int64  `json:"createdAt"`
	UpdatedAt         int64  `json:"updatedAt"`
}

// RecordSummary 是筛选结果的汇总，金额单位为分。
type RecordSummary struct {
	Count   int64 `json:"count"`
	InCent  int64 `json:"inCent"`
	OutCent int64 `json:"outCent"`
}

// RecordStat 是 StatRecord 的一行分组结果；GroupID 是分组列的值，不分组时为 0。
type RecordStat struct {
	GroupID     int64  `gorm:"column:group_id" json:"groupId"`
	RecordCount int64  `gorm:"column:record_count" json:"recordCount"`
	InCent      int64  `gorm:"column:in_cent" json:"inCent"`
	OutCent     int64  `gorm:"column:out_cent" json:"outCent"`
	LastOn      string `gorm:"column:last_on" json:"lastOn"`
}
