// Package book 定义礼簿模型、入参与出参。
package book

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

// UniqueID: not generated because 礼簿允许同名，没有稳定业务键。
type Book struct {
	ID int64 `gorm:"primaryKey;column:id" json:"id"`

	OwnerID     int64  `gorm:"column:owner_id" json:"ownerId"`
	Name        string `gorm:"column:name" json:"name"`
	Category    string `gorm:"column:category" json:"category"`
	Description string `gorm:"column:description" json:"description"`

	CreatedAt int64 `gorm:"autoCreateTime:milli;column:created_at" json:"createdAt"`
	UpdatedAt int64 `gorm:"autoUpdateTime:milli;column:updated_at" json:"updatedAt"`
}

func (vi *Book) TableName() string { return "book" }

func (vi *Book) Serialize() *Book {
	if vi == nil {
		return &Book{}
	}
	vi.Name = strings.TrimSpace(vi.Name)
	vi.Category = strings.TrimSpace(vi.Category)
	vi.Description = strings.TrimSpace(vi.Description)
	if vi.Category == "" {
		vi.Category = consts.BookCategoryRenqing
	}
	return vi
}

func (vi *Book) Deserialize() *Book {
	if vi == nil {
		return &Book{}
	}
	vi.CreatedAt = utils.MilliTimestamp(vi.CreatedAt)
	vi.UpdatedAt = utils.MilliTimestamp(vi.UpdatedAt)
	return vi
}

func (vi *Book) Check() error {
	if vi == nil {
		return fmt.Errorf("礼簿数据为空")
	}
	if vi.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if vi.Name == "" {
		return fmt.Errorf("礼簿名称不能为空")
	}
	if utf8.RuneCountInString(vi.Name) > consts.BookNameMaxLen {
		return fmt.Errorf("礼簿名称不能超过 %d 个字", consts.BookNameMaxLen)
	}
	if utf8.RuneCountInString(vi.Category) > consts.BookCategoryMaxLen {
		return fmt.Errorf("礼簿种类不能超过 %d 个字", consts.BookCategoryMaxLen)
	}
	return nil
}

// ToUpdater 供完整更新使用：种类为空时由 Serialize 补默认值，必须整体写入。
func (vi *Book) ToUpdater() map[string]any {
	return map[string]any{
		"name":        vi.Name,
		"category":    vi.Category,
		"description": vi.Description,
		"updated_at":  time.Now().UTC().UnixMilli(),
	}
}

type CreateBookParam struct {
	OwnerID     int64  `json:"-"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

func (p *CreateBookParam) Serialize() *CreateBookParam {
	if p == nil {
		return &CreateBookParam{}
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Category = strings.TrimSpace(p.Category)
	p.Description = strings.TrimSpace(p.Description)
	return p
}

func (p *CreateBookParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.Name == "" {
		return fmt.Errorf("礼簿名称不能为空")
	}
	return nil
}

// UpdateBookParam 是完整更新：id 来自 query，其余字段来自 body。
type UpdateBookParam struct {
	ID          int64  `json:"-"`
	OwnerID     int64  `json:"-"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

func (p *UpdateBookParam) Serialize() *UpdateBookParam {
	if p == nil {
		return &UpdateBookParam{}
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Category = strings.TrimSpace(p.Category)
	p.Description = strings.TrimSpace(p.Description)
	return p
}

func (p *UpdateBookParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少礼簿 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if p.Name == "" {
		return fmt.Errorf("礼簿名称不能为空")
	}
	return nil
}

// GetBookParam 用于详情与删除。
type GetBookParam struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"-"`
}

func (p *GetBookParam) Serialize() *GetBookParam {
	if p == nil {
		return &GetBookParam{}
	}
	return p
}

func (p *GetBookParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少礼簿 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	return nil
}

// SearchBookParam：Keyword 匹配名称与备注，Category 为精确匹配。
type SearchBookParam struct {
	OwnerID  int64               `json:"-"`
	BookIDs  []int64             `json:"bookIds"`
	Category string              `json:"category"`
	Keyword  string              `json:"keyword"`
	Filter   *modelfilter.Filter `json:"filter"`
}

func (p *SearchBookParam) Serialize() *SearchBookParam {
	if p == nil {
		return &SearchBookParam{Filter: &modelfilter.Filter{}}
	}
	p.Category = strings.TrimSpace(p.Category)
	p.Keyword = strings.TrimSpace(p.Keyword)
	if p.BookIDs == nil {
		p.BookIDs = []int64{}
	}
	if p.Filter == nil {
		p.Filter = &modelfilter.Filter{}
	}
	noSort := p.Filter.SortField == "" && len(p.Filter.OrderByColumns) == 0
	if noSort {
		p.Filter.OrderByColumns = []clause.OrderByColumn{
			{Column: clause.Column{Name: "created_at"}, Desc: true},
			{Column: clause.Column{Name: "id"}, Desc: true},
		}
	}
	return p
}

func (p *SearchBookParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if !consts.BookSortFields[p.Filter.SortField] {
		return fmt.Errorf("不支持按 %s 排序", p.Filter.SortField)
	}
	return nil
}

// BookView 是礼簿的读出口，附带记录合计，金额单位为分。
type BookView struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	CategoryLabel string `json:"categoryLabel"`
	Description   string `json:"description"`
	RecordCount   int64  `json:"recordCount"`
	InCent        int64  `json:"inCent"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}
