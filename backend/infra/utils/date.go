package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Date 是只精确到日的日期，数据库里固定存成 YYYY-MM-DD 文本，字典序即时间序。
type Date struct {
	time.Time
}

func DateOf(t time.Time) Date {
	return Date{Time: NormalizeDate(t, "day")}
}

func DatePtr(t time.Time) *Date {
	d := DateOf(t)
	return &d
}

func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.Format(time.DateOnly), nil
}

func (d *Date) Scan(src any) error {
	if d == nil {
		return fmt.Errorf("utils.Date: Scan on nil receiver")
	}
	if src == nil {
		d.Time = time.Time{}
		return nil
	}
	switch value := src.(type) {
	case time.Time:
		d.Time = NormalizeDate(value, "day")
		return nil
	case string:
		return d.parse(value)
	case []byte:
		return d.parse(string(value))
	default:
		return fmt.Errorf("utils.Date: unsupported source %T", src)
	}
}

func (d *Date) parse(value string) error {
	if value == "" {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := time.ParseInLocation(time.DateOnly, value, time.Local)
	if err != nil {
		// 兼容旧驱动写入的完整时间字符串。
		if fallback, fallbackErr := time.ParseInLocation(time.RFC3339Nano, value, time.Local); fallbackErr == nil {
			d.Time = NormalizeDate(fallback, "day")
			return nil
		}
		return fmt.Errorf("utils.Date: cannot parse %q", value)
	}
	d.Time = parsed
	return nil
}

func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Format(time.DateOnly)
}

// DateStringPtr 用于 *utils.Date 的展示。
func DateStringOf(d *Date) string {
	if d == nil || d.IsZero() {
		return ""
	}
	return d.Format(time.DateOnly)
}
