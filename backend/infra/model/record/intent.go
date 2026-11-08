package record

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"harmony/infra/utils"
)

const (
	IntentText        = "text"
	IntentAmount      = "amount"
	IntentAmountRange = "amount_range"
	IntentYear        = "year"
	IntentYearAmount  = "year_amount"
	IntentPage        = "page"
)

var (
	intentRange    = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(?:-|~|～|—|至)\s*(\d+(?:\.\d+)?)$`)
	intentAmount   = regexp.MustCompile(`^[¥￥]?\s*(\d+(?:\.\d+)?)\s*(?:元|块)?$`)
	intentYear     = regexp.MustCompile(`^(\d{2,4})\s*年$`)
	intentYearDot  = regexp.MustCompile(`^(\d{2})\s*[.．]$`)
	intentPage     = regexp.MustCompile(`^(?:p|P|第)\s*(\d+(?:-\d+)?)\s*页?$`)
	chineseAmount  = regexp.MustCompile(`^[零〇一二两三四五六七八九十百千万萬壹贰叁肆伍陆柒捌玖拾佰仟]+(?:元|块)?$`)
)

// ParseIntent 识别搜索关键词的意图，所有列表接口共用同一套规则。
func ParseIntent(raw string) Intent {
	query := strings.TrimSpace(raw)
	if query == "" {
		return Intent{Kind: IntentText}
	}

	if matches := intentPage.FindStringSubmatch(query); len(matches) == 2 {
		return Intent{Kind: IntentPage, PaperPage: matches[1]}
	}
	if matches := intentRange.FindStringSubmatch(query); len(matches) == 3 {
		min, errMin := utils.ParseYuanToCent(matches[1])
		max, errMax := utils.ParseYuanToCent(matches[2])
		if errMin == nil && errMax == nil {
			if min > max {
				min, max = max, min
			}
			return Intent{Kind: IntentAmountRange, AmountMinCent: min, AmountMaxCent: max}
		}
	}
	if chineseAmount.MatchString(query) {
		body := strings.TrimSuffix(strings.TrimSuffix(query, "元"), "块")
		if number, ok := utils.ParseChineseNumber(body); ok {
			return Intent{Kind: IntentAmount, AmountCent: number * 100, Text: query}
		}
	}
	if matches := intentAmount.FindStringSubmatch(query); len(matches) == 2 {
		cent, err := utils.ParseYuanToCent(matches[1])
		if err == nil {
			// 四位数且落在年份区间时，同时按金额与年份命中。
			if len(strings.TrimLeft(matches[1], "0")) == 4 {
				year, errYear := strconv.ParseInt(matches[1], 10, 64)
				currentYear := int64(time.Now().Year())
				if errYear == nil && year >= 1950 && year <= currentYear {
					return Intent{Kind: IntentYearAmount, AmountCent: cent, Year: year, Text: query}
				}
			}
			return Intent{Kind: IntentAmount, AmountCent: cent, Text: query}
		}
	}
	if matches := intentYear.FindStringSubmatch(query); len(matches) == 2 {
		return Intent{Kind: IntentYear, Year: CompleteYear(matches[1]), Text: query}
	}
	if matches := intentYearDot.FindStringSubmatch(query); len(matches) == 2 {
		return Intent{Kind: IntentYear, Year: CompleteYear(matches[1]), Text: query}
	}
	return Intent{Kind: IntentText, Text: query}
}

// CompleteYear 补全两位年份：不大于当前年份后两位视为 20xx，否则 19xx。
func CompleteYear(value string) int64 {
	year, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	if len(value) >= 4 {
		return year
	}
	current := int64(time.Now().Year())
	if year <= current%100 {
		return 2000 + year
	}
	return 1900 + year
}

// MatchField 返回记录命中的字段，供前端决定高亮位置。
func MatchField(rec *Record, intent Intent) string {
	switch intent.Kind {
	case IntentAmount, IntentAmountRange, IntentYearAmount:
		return "amount"
	case IntentYear:
		return "date"
	case IntentPage:
		return "paperPage"
	}
	text := intent.Text
	if text == "" {
		return "text"
	}
	switch {
	case strings.Contains(rec.PaperNote, text):
		return "paperNote"
	case strings.Contains(rec.GiftName, text):
		return "giftName"
	case strings.Contains(rec.Remark, text):
		return "remark"
	default:
		return "text"
	}
}
