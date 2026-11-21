package utils

import (
	"fmt"
	"strings"
	"time"
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
