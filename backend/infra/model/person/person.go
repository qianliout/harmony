// Package person 定义亲友模型、入参与出参。
package person

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

// UniqueID: not generated because 姓名与备注的唯一性由 uk_person_owner_name_remark 约束，且两者都允许修改。
type Person struct {
	ID int64 `gorm:"primaryKey;column:id" json:"id"`

	OwnerID      int64  `gorm:"column:owner_id" json:"ownerId"`
	Name         string `gorm:"column:name" json:"name"`
	Remark       string `gorm:"column:remark" json:"remark"`
	NamePinyin   string `gorm:"column:name_pinyin" json:"-"`
	NameInitials string `gorm:"column:name_initials" json:"-"`

	CreatedAt int64 `gorm:"autoCreateTime:milli;column:created_at" json:"createdAt"`
	UpdatedAt int64 `gorm:"autoUpdateTime:milli;column:updated_at" json:"updatedAt"`
}

func (vi *Person) TableName() string { return "person" }

func (vi *Person) Serialize() *Person {
	if vi == nil {
		return &Person{}
	}
	vi.Name = strings.TrimSpace(vi.Name)
	vi.Remark = strings.TrimSpace(vi.Remark)
	vi.NamePinyin, vi.NameInitials = utils.Pinyin(vi.Name)
	return vi
}

func (vi *Person) Deserialize() *Person {
	if vi == nil {
		return &Person{}
	}
	vi.CreatedAt = utils.MilliTimestamp(vi.CreatedAt)
	vi.UpdatedAt = utils.MilliTimestamp(vi.UpdatedAt)
	return vi
}

func (vi *Person) Check() error {
	if vi == nil {
		return fmt.Errorf("亲友数据为空")
	}
	if vi.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if vi.Name == "" {
		return fmt.Errorf("姓名不能为空")
	}
	if utf8.RuneCountInString(vi.Name) > consts.PersonNameMaxLen {
		return fmt.Errorf("姓名不能超过 %d 个字", consts.PersonNameMaxLen)
	}
	if utf8.RuneCountInString(vi.Remark) > consts.PersonRemarkMaxLen {
		return fmt.Errorf("备注不能超过 %d 个字", consts.PersonRemarkMaxLen)
	}
	return nil
}

// ToUpdater 供改名使用：拼音字段由姓名派生，必须与姓名一起写入。
func (vi *Person) ToUpdater() map[string]any {
	return map[string]any{
		"name":          vi.Name,
		"remark":        vi.Remark,
		"name_pinyin":   vi.NamePinyin,
		"name_initials": vi.NameInitials,
		"updated_at":    time.Now().UTC().UnixMilli(),
	}
}

func (vi *Person) Same(after *Person) bool {
	if vi == nil || after == nil {
		return false
	}
	return vi.Name == after.Name && vi.Remark == after.Remark
}

type CreatePersonParam struct {
	OwnerID int64  `json:"-"`
	Name    string `json:"name"`
	Remark  string `json:"remark"`
}

func (p *CreatePersonParam) Serialize() *CreatePersonParam {
	if p == nil {
		return &CreatePersonParam{}
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Remark = strings.TrimSpace(p.Remark)
	return p
}

func (p *CreatePersonParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.Name == "" {
		return fmt.Errorf("姓名不能为空")
	}
	return nil
}

// UpdatePersonParam 是完整更新：id 来自 query，其余字段来自 body。
type UpdatePersonParam struct {
	ID      int64  `json:"-"`
	OwnerID int64  `json:"-"`
	Name    string `json:"name"`
	Remark  string `json:"remark"`
}

func (p *UpdatePersonParam) Serialize() *UpdatePersonParam {
	if p == nil {
		return &UpdatePersonParam{}
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Remark = strings.TrimSpace(p.Remark)
	return p
}

func (p *UpdatePersonParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少亲友 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.Name == "" {
		return fmt.Errorf("姓名不能为空")
	}
	return nil
}

// GetPersonParam 用于详情与删除。
type GetPersonParam struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"-"`
}

func (p *GetPersonParam) Serialize() *GetPersonParam {
	if p == nil {
		return &GetPersonParam{}
	}
	return p
}

func (p *GetPersonParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少亲友 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	return nil
}

// MatchPersonParam 按 id 或姓名找到唯一亲友；PersonID 为正时优先按 id。
type MatchPersonParam struct {
	OwnerID      int64  `json:"-"`
	PersonID     int64  `json:"personId"`
	PersonName   string `json:"personName"`
	PersonRemark string `json:"personRemark"`
}

func (p *MatchPersonParam) Serialize() *MatchPersonParam {
	if p == nil {
		return &MatchPersonParam{}
	}
	p.PersonName = strings.TrimSpace(p.PersonName)
	p.PersonRemark = strings.TrimSpace(p.PersonRemark)
	return p
}

func (p *MatchPersonParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.PersonID <= 0 && p.PersonName == "" {
		return fmt.Errorf("请选择或填写亲友")
	}
	return nil
}

// SearchPersonParam：Name/Names 为精确匹配，Keyword 匹配姓名、备注、全拼与首字母。
type SearchPersonParam struct {
	OwnerID   int64               `json:"-"`
	PersonIDs []int64             `json:"personIds"`
	Name      string              `json:"name"`
	Names     []string            `json:"names"`
	Keyword   string              `json:"keyword"`
	Filter    *modelfilter.Filter `json:"filter"`
}

func (p *SearchPersonParam) Serialize() *SearchPersonParam {
	if p == nil {
		return &SearchPersonParam{Filter: &modelfilter.Filter{}}
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Keyword = strings.ToLower(strings.TrimSpace(p.Keyword))
	if p.PersonIDs == nil {
		p.PersonIDs = []int64{}
	}
	p.Names = utils.NonEmptyStrings(utils.TrimStrings(p.Names)...)
	if p.Filter == nil {
		p.Filter = &modelfilter.Filter{}
	}
	noSort := p.Filter.SortField == "" && len(p.Filter.OrderByColumns) == 0
	if noSort {
		p.Filter.OrderByColumns = []clause.OrderByColumn{
			{Column: clause.Column{Name: "updated_at"}, Desc: true},
			{Column: clause.Column{Name: "id"}, Desc: true},
		}
	}
	return p
}

func (p *SearchPersonParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if !consts.PersonSortFields[p.Filter.SortField] {
		return fmt.Errorf("不支持按 %s 排序", p.Filter.SortField)
	}
	return nil
}

// PersonView 是亲友的读出口，附带往来合计，金额单位为分。
type PersonView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Remark      string `json:"remark"`
	DisplayName string `json:"displayName"`
	InCent      int64  `json:"inCent"`
	OutCent     int64  `json:"outCent"`
	RecordCount int64  `json:"recordCount"`
	LastOn      string `json:"lastOn"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// DisplayName 把姓名与备注拼成「张三 · 村东」，备注为空时只显示姓名。
func (vi *Person) DisplayName() string {
	if vi.Remark == "" {
		return vi.Name
	}
	return vi.Name + consts.PersonDisplaySep + vi.Remark
}
