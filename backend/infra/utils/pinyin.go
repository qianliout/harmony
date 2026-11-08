package utils

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// Pinyin 生成姓名全拼与首字母：张三 → zhangsan / zs。
// 非汉字字符按原字符保留，统一小写；空格与标点丢弃。
func Pinyin(name string) (full string, initials string) {
	args := pinyin.NewArgs()
	var fullBuilder, initialsBuilder strings.Builder
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsSpace(r) {
			continue
		}
		syllables := pinyin.LazyPinyin(string(r), args)
		if len(syllables) > 0 && syllables[0] != "" {
			s := strings.ToLower(syllables[0])
			fullBuilder.WriteString(s)
			initialsBuilder.WriteByte(s[0])
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			lower := strings.ToLower(string(r))
			fullBuilder.WriteString(lower)
			if unicode.IsLetter(r) {
				initialsBuilder.WriteString(lower)
			}
		}
	}
	return fullBuilder.String(), initialsBuilder.String()
}
