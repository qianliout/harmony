package utils

import (
	"fmt"
	"strings"
	"time"

	"harmony/infra/consts"
)

const DateLayout = "2006-01-02"

// ParseDate 严格解析 YYYY-MM-DD。
func ParseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("date is empty")
	}
	t, err := time.ParseInLocation(DateLayout, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q", value)
	}
	return t, nil
}

func DateString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateLayout)
}

func DateStringPtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(DateLayout)
}

func Millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func MillisPtr(t *time.Time) int64 {
	if t == nil || t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func NowMillis() int64 {
	return time.Now().UnixMilli()
}

// NormalizeDate 按精度归到该月或该年第一天；日期只到月/年时按文档口径存储。
func NormalizeDate(t time.Time, precision string) time.Time {
	switch precision {
	case consts.DatePrecisionMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case consts.DatePrecisionYear:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
}

// AddYears 保留月末语义地加年，用于年份切换。
func YearRange(year int64) (time.Time, time.Time) {
	start := time.Date(int(year), 1, 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(1, 0, 0)
}
