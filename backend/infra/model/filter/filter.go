package model

import (
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harmony/infra/consts"
)

type Filter struct {
	// SortBy/SortField 表示单列排序。
	SortBy    string `json:"sortBy"`
	SortField string `json:"sortField"`

	// OrderByColumns 表示多列或显式排序，在单列排序之后追加。
	OrderByColumns []clause.OrderByColumn

	Offset int64 `json:"offset"`
	Limit  int64 `json:"limit"`
}

func (vi *Filter) SetSortCreatedAt() *Filter {
	af := vi.DeepCopy()
	af.SortField = consts.FilterSortFieldCreatedAt
	return af
}

func (vi *Filter) SetSortDesc() *Filter {
	af := vi.DeepCopy()
	af.SortBy = consts.FilterSortDesc
	return af
}

func (vi *Filter) SetSortAsc() *Filter {
	af := vi.DeepCopy()
	af.SortBy = consts.FilterSortAsc
	return af
}

func (vi *Filter) SetSortFieldByID() *Filter {
	af := vi.DeepCopy()
	af.SortField = consts.FilterSortFieldID
	return af
}

func (vi *Filter) SetLimit(limit int64) *Filter {
	af := vi.DeepCopy()
	af.Limit = limit
	return af
}

func (vi *Filter) SetMaxLimit(limit int64) *Filter {
	af := vi.DeepCopy()
	outOfRange := af.Limit > limit || af.Limit <= 0
	if outOfRange {
		af.Limit = limit
	}
	return af
}

func (vi *Filter) SetOffset(offset int64) *Filter {
	af := vi.DeepCopy()
	af.Offset = offset
	return af
}

func (vi *Filter) SetSortField(sortField string) *Filter {
	af := vi.DeepCopy()
	af.SortField = sortField
	return af
}

func EmptyFilter() *Filter {
	return &Filter{}
}

// GetFilter 未传 limit 时不限条数，返回全部行。
func GetFilter(ctx *gin.Context) *Filter {
	offset, _ := strconv.ParseInt(ctx.Query(consts.FilterQueryOffset), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query(consts.FilterQueryLimit), 10, 64)
	sortBy := ctx.Query(consts.FilterQuerySortBy)
	sortField := ctx.Query(consts.FilterQuerySortField)
	if limit <= 0 {
		limit = math.MaxInt32
	}
	if offset <= 0 {
		offset = 0
	}

	filter := &Filter{Offset: offset, Limit: limit, SortBy: sortBy, SortField: sortField}
	filter.OrderByColumns = make([]clause.OrderByColumn, 0)
	return filter
}

func GetFilterWithDefault(ctx *gin.Context) *Filter {
	offset, _ := strconv.ParseInt(ctx.Query(consts.FilterQueryOffset), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query(consts.FilterQueryLimit), 10, 64)
	sortBy := ctx.Query(consts.FilterQuerySortBy)
	sortField := ctx.Query(consts.FilterQuerySortField)

	filter := &Filter{Offset: offset, Limit: limit, SortBy: sortBy, SortField: sortField}
	filter.OrderByColumns = make([]clause.OrderByColumn, 0)
	filter = filter.SetDefault()
	return filter
}

func EmptyFilterForTotalQuery() *Filter {
	return &Filter{
		Offset:    0,
		Limit:     math.MaxInt32,
		SortBy:    consts.FilterSortDesc,
		SortField: consts.FilterSortFieldID,
	}
}

func (vi *Filter) SetDefault() *Filter {
	if vi == nil {
		return &Filter{
			Offset:    0,
			Limit:     math.MaxInt64,
			SortBy:    consts.FilterSortDesc,
			SortField: consts.FilterSortFieldID,
		}
	}
	vi.SortBy = strings.ToLower(vi.SortBy)
	validSortBy := vi.SortBy == consts.FilterSortDesc || vi.SortBy == consts.FilterSortAsc
	if !validSortBy {
		vi.SortBy = consts.FilterSortDesc
	}
	if vi.Offset <= 0 {
		vi.Offset = 0
	}
	if vi.Limit <= 0 {
		vi.Limit = consts.FilterDefaultLimit
	}
	return vi
}

// DeepCopy 不复制 OrderByColumns。
func (vi *Filter) DeepCopy() *Filter {
	if vi == nil {
		return &Filter{
			Limit: math.MaxInt32,
		}
	}
	cp := Filter{
		SortBy:    vi.SortBy,
		SortField: vi.SortField,
		Offset:    vi.Offset,
		Limit:     vi.Limit,
	}
	return &cp
}

// AddFilter 收到 nil filter 时不加分页与排序：项目里有多处流程有意查全表。
func AddFilter(db *gorm.DB, filter *Filter) *gorm.DB {
	if filter == nil {
		return db
	}
	hasPage := filter.Offset >= 0 && filter.Limit > 0
	if hasPage {
		db = db.Offset(int(filter.Offset))
		db = db.Limit(int(filter.Limit))
	}
	hasSort := filter.SortField != "" && filter.SortBy != ""
	if hasSort {
		desc := strings.ToLower(filter.SortBy) == consts.FilterSortDesc
		column := clause.OrderByColumn{Column: clause.Column{Name: filter.SortField}, Desc: desc}
		db = db.Order(column)
	}
	for i := range filter.OrderByColumns {
		db = db.Order(filter.OrderByColumns[i])
	}
	return db
}

func AddFilterWithDefault(db *gorm.DB, filter *Filter) *gorm.DB {
	if filter == nil {
		filter = &Filter{Limit: consts.FilterNilLimit}
	}
	filtered := AddFilter(db, filter)
	return filtered
}
