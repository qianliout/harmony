package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseYuanToCent 用字符串拆分元与角分，避免浮点误差。
func ParseYuanToCent(value string) (int64, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	value = strings.TrimPrefix(value, "¥")
	value = strings.TrimPrefix(value, "￥")
	value = strings.TrimSuffix(value, "元")
	if value == "" {
		return 0, fmt.Errorf("amount is empty")
	}
	parts := strings.SplitN(value, ".", 2)
	yuanPart := parts[0]
	if yuanPart == "" {
		yuanPart = "0"
	}
	yuan, err := strconv.ParseInt(yuanPart, 10, 64)
	if err != nil || yuan < 0 {
		return 0, fmt.Errorf("invalid amount %q", value)
	}
	fen := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > 2 {
			fraction = fraction[:2]
		}
		for len(fraction) < 2 {
			fraction += "0"
		}
		if fraction != "" {
			parsed, err := strconv.ParseInt(fraction, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid amount %q", value)
			}
			fen = parsed
		}
	}
	return yuan*100 + fen, nil
}

// YuanString 把分转成两位小数的元字符串，用于导出。
func YuanString(cent int64) string {
	negative := cent < 0
	if negative {
		cent = -cent
	}
	body := fmt.Sprintf("%d.%02d", cent/100, cent%100)
	if negative {
		return "-" + body
	}
	return body
}
