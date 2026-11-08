package utils

import "strings"

var chineseDigits = map[rune]int64{
	'零': 0, '〇': 0, '一': 1, '二': 2, '两': 2, '三': 3, '四': 4,
	'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	'壹': 1, '贰': 2, '叁': 3, '肆': 4, '伍': 5, '陆': 6, '柒': 7, '捌': 8, '玖': 9,
}

var chineseUnits = map[rune]int64{
	'十': 10, '拾': 10, '百': 100, '佰': 100, '千': 1000, '仟': 1000,
	'万': 10000, '萬': 10000, '亿': 100000000,
}

// ParseChineseNumber 解析"伍拾""壹佰贰拾""一百零五""两千"这类中文数字。
func ParseChineseNumber(text string) (int64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	var total, section, number int64
	sawAny := false
	for _, r := range text {
		if digit, ok := chineseDigits[r]; ok {
			number = digit
			sawAny = true
			continue
		}
		unit, ok := chineseUnits[r]
		if !ok {
			return 0, false
		}
		sawAny = true
		if unit >= 10000 {
			section += number
			if section == 0 {
				section = 1
			}
			total += section * unit
			section, number = 0, 0
			continue
		}
		if number == 0 {
			number = 1
		}
		section += number * unit
		number = 0
	}
	if !sawAny {
		return 0, false
	}
	return total + section + number, true
}
