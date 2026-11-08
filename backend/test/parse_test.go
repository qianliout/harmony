package test

import (
	"testing"

	recordservice "harmony/component/record/service"
	"harmony/infra/consts"
	"harmony/infra/model/record"
)

func singleDraft(t *testing.T, result record.PaperParseResult) record.PaperDraft {
	t.Helper()
	if result.OnlyMark {
		t.Fatalf("unexpected onlyMark result")
	}
	if len(result.Repays) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(result.Repays))
	}
	return result.Repays[0]
}

func itemCents(draft record.PaperDraft) map[string]int64 {
	out := make(map[string]int64, len(draft.Items))
	for _, item := range draft.Items {
		out[item.ItemKind] += item.AmountCent
	}
	return out
}

// TestParseRuleTable 对应 docs/02「标注解析」：
// 表驱动覆盖解析规则表每一行（年份、具体日期、节令、只有年、名目金额、现金金额、事项、已还记号），
// 外加「10年8月1日 生日 500」「11年正月 办儿 50元」「11年腊月 办儿 50元」三个真实样例。
func TestParseRuleTable(t *testing.T) {
	type checkFunc func(t *testing.T, draft record.PaperDraft)
	dateCase := func(wantDate, wantPrecision string) checkFunc {
		return func(t *testing.T, draft record.PaperDraft) {
			if draft.OccurredOn != wantDate {
				t.Fatalf("date: want %q, got %q", wantDate, draft.OccurredOn)
			}
			if draft.DatePrecision != wantPrecision {
				t.Fatalf("precision: want %q, got %q", wantPrecision, draft.DatePrecision)
			}
		}
	}
	itemCase := func(kind string, wantCent int64) checkFunc {
		return func(t *testing.T, draft record.PaperDraft) {
			cents := itemCents(draft)
			if cents[kind] != wantCent {
				t.Fatalf("item %s: want %d, got %d (all=%v)", kind, wantCent, cents[kind], cents)
			}
		}
	}
	eventCase := func(name string) checkFunc {
		return func(t *testing.T, draft record.PaperDraft) {
			if draft.Event == nil || draft.Event.Name != name {
				t.Fatalf("event: want %q, got %+v", name, draft.Event)
			}
		}
	}

	cases := []struct {
		name  string
		note  string
		check checkFunc
	}{
		{"年份-两位", "11年", dateCase("2011-01-01", consts.DatePrecisionYear)},
		{"年份-四位", "2011年", dateCase("2011-01-01", consts.DatePrecisionYear)},
		{"年份-点号", "11.", dateCase("2011-01-01", consts.DatePrecisionYear)},
		{"具体日期-月日", "11年8月1日", dateCase("2011-08-01", consts.DatePrecisionDay)},
		{"节令-国庆", "11年国庆", dateCase("2011-10-01", consts.DatePrecisionMonth)},
		{"节令-正月", "11年正月", dateCase("2011-02-01", consts.DatePrecisionMonth)},
		{"节令-腊月跨年", "11年腊月", dateCase("2012-01-01", consts.DatePrecisionMonth)},
		{"节令-中秋", "11年中秋", dateCase("2011-09-01", consts.DatePrecisionMonth)},
		{"名目金额-火炮", "火炮20元", itemCase(consts.ItemKindHuopao, 2000)},
		{"名目金额-炮", "炮20", itemCase(consts.ItemKindHuopao, 2000)},
		{"现金金额-元", "100元", itemCase(consts.ItemKindCash, 10000)},
		{"现金金额-裸数字", "100", itemCase(consts.ItemKindCash, 10000)},
		{"现金金额-大写", "壹佰元", itemCase(consts.ItemKindCash, 10000)},
		{"事项-剩余中文", "11年国庆 孙女 100元", eventCase("孙女")},
		{"真实样例-生日", "10年8月1日 生日 500", func(t *testing.T, draft record.PaperDraft) {
			dateCase("2010-08-01", consts.DatePrecisionDay)(t, draft)
			itemCase(consts.ItemKindCash, 50000)(t, draft)
			eventCase("生日")(t, draft)
			if draft.Confidence != "high" {
				t.Fatalf("confidence: want high, got %q", draft.Confidence)
			}
		}},
		{"真实样例-办儿", "11年正月 办儿 50元", func(t *testing.T, draft record.PaperDraft) {
			dateCase("2011-02-01", consts.DatePrecisionMonth)(t, draft)
			itemCase(consts.ItemKindCash, 5000)(t, draft)
			eventCase("办儿")(t, draft)
			if draft.Event.Type != consts.EventTypeFullMoon {
				t.Fatalf("event type: want fullmoon, got %q", draft.Event.Type)
			}
		}},
		{"真实样例-办儿腊月", "11年腊月 办儿 50元", func(t *testing.T, draft record.PaperDraft) {
			dateCase("2012-01-01", consts.DatePrecisionMonth)(t, draft)
			itemCase(consts.ItemKindCash, 5000)(t, draft)
			eventCase("办儿")(t, draft)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := recordservice.ParsePaperNoteText(tc.note)
			draft := singleDraft(t, result)
			tc.check(t, draft)
		})
	}

	// 已还记号单独断言 onlyMark，不产生草稿。
	for _, mark := range []string{"已还", "✓", "√", "还了"} {
		t.Run("已还记号-"+mark, func(t *testing.T) {
			result := recordservice.ParsePaperNoteText(mark)
			if !result.OnlyMark {
				t.Fatalf("note %q: expected onlyMark=true", mark)
			}
			if len(result.Repays) != 0 {
				t.Fatalf("note %q: expected no drafts, got %d", mark, len(result.Repays))
			}
		})
	}
}

// TestParseShorthandMonthDay 对应 docs/02 解析规则表「具体日期 `8.1` → 08-01，精度 day」。
// 当前生产解析器把「11年8.1」的「8.1」当成 8.10 元现金，日期也只到 2011 年。
// 保留该失败断言用于暴露缺陷，不弱化预期。
func TestParseShorthandMonthDay(t *testing.T) {
	result := recordservice.ParsePaperNoteText("11年8.1")
	draft := singleDraft(t, result)
	if draft.OccurredOn != "2011-08-01" {
		t.Fatalf("expected shorthand date 2011-08-01, got %q (precision %q)", draft.OccurredOn, draft.DatePrecision)
	}
	if draft.DatePrecision != consts.DatePrecisionDay {
		t.Fatalf("expected day precision, got %q", draft.DatePrecision)
	}
	if cents := itemCents(draft); cents[consts.ItemKindCash] != 0 {
		t.Fatalf("expected no cash item from date shorthand, got %v", cents)
	}
}
