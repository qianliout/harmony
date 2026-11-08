package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"harmony/infra/consts"
	"harmony/infra/model/event"
	"harmony/infra/model/record"
	"harmony/infra/utils"
)

// 纸本标注解析：正则加规则，不调用大模型，两端共用同一口径。
var (
	paperSplit   = regexp.MustCompile(`[\n\r;；/／]+`)
	paperYear    = regexp.MustCompile(`(\d{2,4})\s*年`)
	paperYearDot = regexp.MustCompile(`^\s*(\d{2})\s*[.．]`)
	paperMonth   = regexp.MustCompile(`(\d{1,2})\s*月(?:\s*(\d{1,2})\s*[日号])?`)
	paperMonthDot = regexp.MustCompile(`\b(\d{1,2})\s*[.．]\s*(\d{1,2})\b`)
	paperHuopao  = regexp.MustCompile(`(?:火炮|大炮|炮)\s*([0-9]+(?:\.[0-9]+)?|[零〇一二两三四五六七八九十百千万萬壹贰叁肆伍陆柒捌玖拾佰仟]+)\s*元?`)
	paperAmount  = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?|[零〇一二两三四五六七八九十百千万萬壹贰叁肆伍陆柒捌玖拾佰仟]+)\s*(?:元|块)?`)
	paperClean   = regexp.MustCompile(`[\s，。,.、·:：;；!！?？()（）\[\]【】\-—_]+`)
	paperMark    = regexp.MustCompile(`已还|还了|已回|還|✓|√|✔`)
)

var festivals = []struct {
	keyword string
	month   int
	nextYear bool
}{
	{"正月", 2, false},
	{"二月二", 2, false},
	{"五一", 5, false},
	{"端午", 6, false},
	{"中秋", 9, false},
	{"国庆", 10, false},
	{"腊月", 1, true},
	{"春节", 2, false},
	{"过年", 2, false},
}

// ParsePaperNoteText 供识别链路复用同一套标注解析规则。
func ParsePaperNoteText(note string) record.PaperParseResult {
	return parsePaperNote(note)
}

// parsePaperNote 把一条手写标注拆成多次还礼草稿。
func parsePaperNote(note string) record.PaperParseResult {
	result := record.PaperParseResult{Repays: make([]record.PaperDraft, 0)}
	note = strings.TrimSpace(note)
	if note == "" {
		return result
	}

	segments := paperSplit.Split(note, -1)
	expanded := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		// 同一行里出现第二处年份时（常见于两行标注挤在一起），在年份前切开。
		for _, part := range splitAtYearStarts(segment) {
			part = strings.TrimSpace(part)
			if part != "" {
				expanded = append(expanded, part)
			}
		}
	}

	markSeen := false
	drafts := make([]record.PaperDraft, 0, len(expanded))
	for _, segment := range expanded {
		draft, onlyMark := parsePaperSegment(segment)
		if onlyMark {
			markSeen = true
			continue
		}
		drafts = append(drafts, draft)
	}
	if len(drafts) == 0 && markSeen {
		result.OnlyMark = true
	}
	if len(drafts) > 0 {
		result.Repays = drafts
	}
	return result
}

func splitAtYearStarts(segment string) []string {
	indices := paperYear.FindAllStringIndex(segment, -1)
	if len(indices) <= 1 {
		return []string{segment}
	}
	parts := make([]string, 0, len(indices))
	start := 0
	for i := 1; i < len(indices); i++ {
		parts = append(parts, segment[start:indices[i][0]])
		start = indices[i][0]
	}
	parts = append(parts, segment[start:])
	return parts
}

func parsePaperSegment(segment string) (record.PaperDraft, bool) {
	segment = strings.TrimSpace(segment)
	draft := record.PaperDraft{
		OccurredOn:    "",
		DatePrecision: consts.DatePrecisionYear,
		Items:         make([]record.RepayItem, 0),
		Raw:           segment,
		Confidence:    "low",
	}
	if segment == "" {
		return draft, false
	}
	if paperMark.MatchString(segment) && !hasAmountDigits(segment) {
		return draft, true
	}

	work := segment

	year := int64(0)
	if matches := paperYear.FindStringSubmatch(work); len(matches) == 2 {
		year = record.CompleteYear(matches[1])
		work = strings.Replace(work, matches[0], " ", 1)
	} else if matches := paperYearDot.FindStringSubmatch(work); len(matches) == 2 {
		year = record.CompleteYear(matches[1])
		work = work[len(matches[0]):]
	}

	month := 0
	day := 0
	if matches := paperMonth.FindStringSubmatch(work); len(matches) >= 2 && matches[1] != "" {
		month, _ = strconv.Atoi(matches[1])
		if len(matches) == 3 && matches[2] != "" {
			day, _ = strconv.Atoi(matches[2])
		}
		work = strings.Replace(work, matches[0], " ", 1)
	}
	// 纸本上常写成 8.1：年份去掉之后再识别月.日，避免被当成金额。
	if month == 0 {
		if matches := paperMonthDot.FindStringSubmatch(work); len(matches) == 3 {
			candidateMonth, _ := strconv.Atoi(matches[1])
			candidateDay, _ := strconv.Atoi(matches[2])
			if candidateMonth >= 1 && candidateMonth <= 12 && candidateDay >= 1 && candidateDay <= 31 {
				month = candidateMonth
				day = candidateDay
				work = strings.Replace(work, matches[0], " ", 1)
			}
		}
	}
	nextYear := false
	if month == 0 {
		for _, festival := range festivals {
			if strings.Contains(work, festival.keyword) {
				month = festival.month
				nextYear = festival.nextYear
				work = strings.ReplaceAll(work, festival.keyword, " ")
				break
			}
		}
	}

	precision := consts.DatePrecisionYear
	if day > 0 {
		precision = consts.DatePrecisionDay
	} else if month > 0 {
		precision = consts.DatePrecisionMonth
	}
	if year > 0 {
		if nextYear {
			year++
		}
		useMonth := month
		if useMonth == 0 {
			useMonth = 1
		}
		useDay := day
		if useDay == 0 {
			useDay = 1
		}
		draft.OccurredOn = utils.DateString(utils.NormalizeDate(
			mustDate(int(year), useMonth, useDay), precision))
		draft.DatePrecision = precision
	} else {
		draft.DatePrecision = precision
	}

	// 火炮等具名金额：先取出并从原文删掉，避免被当成现金。
	for {
		matches := paperHuopao.FindStringSubmatch(work)
		if len(matches) != 2 {
			break
		}
		if cent, ok := paperAmountToCent(matches[1]); ok {
			draft.Items = append(draft.Items, record.RepayItem{
				ItemKind:   consts.ItemKindHuopao,
				AmountCent: cent,
			})
		}
		work = strings.Replace(work, matches[0], " ", 1)
	}

	// 剩余的第一个金额视为现金。
	if matches := paperAmount.FindStringSubmatch(work); len(matches) == 2 && matches[1] != "" {
		if cent, ok := paperAmountToCent(matches[1]); ok {
			draft.Items = append(draft.Items, record.RepayItem{
				ItemKind:   consts.ItemKindCash,
				AmountCent: cent,
			})
		}
		work = strings.Replace(work, matches[0], " ", 1)
	}

	eventName := cleanEventName(work)
	if eventName != "" {
		draft.Event = &record.EventParam{
			Name:          eventName,
			Type:          event.InferTypeByName(eventName),
			Date:          draft.OccurredOn,
			DatePrecision: draft.DatePrecision,
		}
	}
	if draft.OccurredOn != "" && eventName != "" && len(draft.Items) > 0 {
		draft.Confidence = "high"
	}
	draft.Raw = segment
	return draft, false
}

func cleanEventName(work string) string {
	work = paperMark.ReplaceAllString(work, " ")
	replacer := strings.NewReplacer(
		"贺礼", " ", "还礼", " ", "送礼", " ", "收礼", " ", "收", " ",
		"元", " ", "块", " ", "约", " ", "整", " ",
	)
	work = replacer.Replace(work)
	work = paperClean.ReplaceAllString(work, " ")
	fields := strings.Fields(work)
	cleaned := strings.Join(fields, "")
	if utf8.RuneCountInString(cleaned) > 32 {
		cleaned = string([]rune(cleaned)[:32])
	}
	return cleaned
}

func paperAmountToCent(text string) (int64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	if isASCIIDigits(text) {
		cent, err := utils.ParseYuanToCent(text)
		return cent, err == nil
	}
	number, ok := utils.ParseChineseNumber(text)
	if !ok {
		return 0, false
	}
	return number * 100, true
}

func isASCIIDigits(text string) bool {
	for _, r := range text {
		if !unicode.IsDigit(r) && r != '.' {
			return false
		}
	}
	return true
}

func hasAmountDigits(text string) bool {
	for _, r := range text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	for _, r := range text {
		if strings.ContainsRune("零〇一二两三四五六七八九十百千万萬壹贰叁肆伍陆柒捌玖拾佰仟", r) {
			return true
		}
	}
	return false
}

func mustDate(year int, month int, day int) (t time.Time) {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}
