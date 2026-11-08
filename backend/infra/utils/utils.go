package utils

import (
	"cmp"
	"encoding/json"
	"hash/fnv"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
)

func GetInt64FromAny(pre any) int64 {
	if pre == nil {
		return 0
	}
	switch x := pre.(type) {
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0
		}
		return i
	}
	return 0
}

func GetInt64FromQuery(c *gin.Context, key string) int64 {
	v := c.Query(key)
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

// GetInt64sFromQuery 读取逗号分隔的 int64 列表，丢弃非法值与非正数。
func GetInt64sFromQuery(c *gin.Context, key string) []int64 {
	values := GetStrsFromQuery(c, key)
	out := make([]int64, 0, len(values))
	for i := range values {
		v, err := strconv.ParseInt(values[i], 10, 64)
		invalid := err != nil || v <= 0
		if invalid {
			continue
		}
		out = append(out, v)
	}
	return out
}

// GetStrsFromQuery 读取字符串列表：同名参数可重复出现，每个值内也可逗号分隔，空项丢弃。
func GetStrsFromQuery(c *gin.Context, key string) []string {
	emptyKey := strings.TrimSpace(key) == ""
	if c == nil || emptyKey {
		return []string{}
	}

	values := c.QueryArray(key)
	if len(values) == 0 {
		values = strings.Split(c.Query(key), consts.QueryListSep)
	}

	joined := strings.Join(values, consts.QueryListSep)
	parts := strings.Split(joined, consts.QueryListSep)
	out := make([]string, 0, len(parts))
	for i := range parts {
		v := strings.TrimSpace(parts[i])
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func GetStrFromQuery(c *gin.Context, key string) string {
	v := c.Query(key)
	return strings.TrimSpace(v)
}

func GenerateInt64(strs ...string) int64 {
	s := strings.Join(strs, "0x1f")
	h := fnv.New64a()
	// hash.Hash 的 Write 约定永不返回错误。
	_, _ = h.Write([]byte(s))
	result := int64(h.Sum64())
	if result >= 0 {
		return result
	}
	if result == math.MinInt64 {
		return math.MaxInt64
	}
	return -result
}

func GetFilter(c *gin.Context) *modelfilter.Filter {
	page := GetInt64FromQuery(c, consts.QueryPage)
	pageSize := GetInt64FromQuery(c, consts.QueryPageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > consts.MaxPageSize {
		pageSize = consts.DefaultPageSize
	}
	sortBy := GetStrFromQuery(c, consts.QuerySortBy)
	sortField := GetStrFromQuery(c, consts.QuerySortField)

	return &modelfilter.Filter{
		SortBy:    sortBy,
		SortField: sortField,
		Offset:    (page - 1) * pageSize,
		Limit:     pageSize,
	}
}

// DedupeSorted 去重后升序返回，结果始终非 nil。
func DedupeSorted[T cmp.Ordered](items []T) []T {
	if len(items) == 0 {
		return []T{}
	}
	seen := make(map[T]struct{}, len(items))
	out := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	slices.Sort(out)
	return out
}

// NonEmptyStrings 按原顺序丢弃空白串，保留的元素不做 trim，结果始终非 nil。
func NonEmptyStrings(items ...string) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		blank := strings.TrimSpace(s) == ""
		if blank {
			continue
		}
		out = append(out, s)
	}
	return out
}

// TrimStrings 原地 trim 每个元素并返回同一切片。
func TrimStrings(items []string) []string {
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}
	return items
}

// MilliTimestamp 把项目统一毫秒前写入的秒级时间戳转成毫秒，毫秒值原样返回。
func MilliTimestamp(stored int64) int64 {
	if stored > 0 && stored < consts.TimestampMilliThreshold {
		return stored * 1000
	}
	return stored
}
