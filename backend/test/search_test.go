package test

import (
	"testing"

	"harmony/infra/model/record"
	"harmony/infra/model/search"
)

// TestSearchIntent 对应 docs/02「搜索意图」：
// `500` 命中金额 500；`伍佰` 同样；`孙女` 命中纸本标注；`p1-10` 命中该页记录。
func TestSearchIntent(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张光轩")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	target := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
		PaperNote: "孙女", PaperPage: "1-10", PaperSeq: 1,
	})
	// 干扰项：金额、标注、页码都不同。
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 30000, Direction: "in", OccurredOn: "2011-08-20",
		Confirm: true,
	})

	cases := []struct {
		name       string
		query      string
		wantIntent string
		wantField  string
	}{
		{"金额数字", "500", record.IntentAmount, "amount"},
		{"金额大写", "伍佰", record.IntentAmount, "amount"},
		{"纸本标注", "孙女", record.IntentText, "paperNote"},
		{"纸本页码", "p1-10", record.IntentPage, "paperPage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := h.searchSrv.Search(h.ctx(), tc.query, 20)
			requireOK(t, err)
			if result.Intent != tc.wantIntent {
				t.Fatalf("intent: want %q, got %q", tc.wantIntent, result.Intent)
			}
			hit := findRecordHit(t, result, target.ID)
			if hit.MatchField != tc.wantField {
				t.Fatalf("matchField: want %q, got %q", tc.wantField, hit.MatchField)
			}
		})
	}
}

func findRecordHit(t *testing.T, result *search.Result, id uint64) *search.RecordHit {
	t.Helper()
	for i := range result.Records.Items {
		if result.Records.Items[i].ID == id {
			return &result.Records.Items[i]
		}
	}
	t.Fatalf("record %d not found in search results (%d hits)", id, len(result.Records.Items))
	return nil
}
