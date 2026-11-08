package test

import (
	"testing"

	"harmony/infra/consts"
	"harmony/infra/model/book"
	"harmony/infra/model/record"
)

// TestCrossCategoryMerge 对应 docs/02「跨类别合并」：
// 火炮本并入人情礼本后，按名目统计火炮金额不变；origin_book_id 不变；源本子 merged。
func TestCrossCategoryMerge(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	targetBook := h.createBook("人情礼本", "renqing", 0)
	sourceBook := h.createBook("火炮本", "huopao", 0)

	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: targetBook,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})
	moved := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: sourceBook,
		AmountCent: 10000, Direction: "in", OccurredOn: "2011-08-20",
	})

	_, err := h.bookSrv.Merge(h.ctx(), book.MergeParam{
		SourceIDs: []uint64{sourceBook},
		TargetID:  targetBook,
	})
	requireOK(t, err)

	// 名目统计不变：火炮仍然是 10000 分，现金 50000 分，不串项。
	bookStats, err := h.statsSrv.Book(h.ctx(), targetBook)
	requireOK(t, err)
	byItem := make(map[string]int64)
	for _, part := range bookStats.ByItem {
		byItem[part.ItemKind] += part.Cent
	}
	if byItem["huopao"] != 10000 {
		t.Fatalf("expected huopao 10000 after merge, got %d (all=%v)", byItem["huopao"], byItem)
	}
	if byItem["cash"] != 50000 {
		t.Fatalf("expected cash 50000 after merge, got %d (all=%v)", byItem["cash"], byItem)
	}

	// 记录归属目标本子，origin_book_id 保留原本子。
	movedRecord, err := h.recordStore.Get(h.tx, 1, moved.ID)
	requireOK(t, err)
	if movedRecord.BookID != targetBook {
		t.Fatalf("expected moved book %d, got %d", targetBook, movedRecord.BookID)
	}
	if movedRecord.OriginBookID != sourceBook {
		t.Fatalf("expected origin book %d, got %d", sourceBook, movedRecord.OriginBookID)
	}
	if movedRecord.ItemKind != consts.ItemKindHuopao {
		t.Fatalf("expected huopao item kind preserved, got %q", movedRecord.ItemKind)
	}

	source, err := h.bookStore.Get(h.tx, 1, sourceBook)
	requireOK(t, err)
	if source.Status != consts.BookStatusMerged {
		t.Fatalf("expected source merged, got %q", source.Status)
	}
	if source.MergedIntoID == nil || *source.MergedIntoID != targetBook {
		t.Fatalf("expected source merged_into %d, got %v", targetBook, source.MergedIntoID)
	}
}

// TestSecondMergeRepointsChain 对应 docs/02「二次合并」：
// A 并入 B，B 再并入 C，A 的 merged_into_id 变为 C。
func TestSecondMergeRepointsChain(t *testing.T) {
	h := newHarness(t)
	bookA := h.createBook("礼簿 A", "renqing", 0)
	bookB := h.createBook("礼簿 B", "renqing", 0)
	bookC := h.createBook("礼簿 C", "renqing", 0)

	_, err := h.bookSrv.Merge(h.ctx(), book.MergeParam{SourceIDs: []uint64{bookA}, TargetID: bookB})
	requireOK(t, err)
	_, err = h.bookSrv.Merge(h.ctx(), book.MergeParam{SourceIDs: []uint64{bookB}, TargetID: bookC})
	requireOK(t, err)

	a, err := h.bookStore.Get(h.tx, 1, bookA)
	requireOK(t, err)
	if a.Status != consts.BookStatusMerged {
		t.Fatalf("expected A merged, got %q", a.Status)
	}
	if a.MergedIntoID == nil || *a.MergedIntoID != bookC {
		t.Fatalf("expected A merged_into %d (C), got %v", bookC, a.MergedIntoID)
	}
	b, err := h.bookStore.Get(h.tx, 1, bookB)
	requireOK(t, err)
	if b.Status != consts.BookStatusMerged || b.MergedIntoID == nil || *b.MergedIntoID != bookC {
		t.Fatalf("expected B merged into C, got status=%q mergedInto=%v", b.Status, b.MergedIntoID)
	}
	c, err := h.bookStore.Get(h.tx, 1, bookC)
	requireOK(t, err)
	if c.Status != consts.BookStatusNormal {
		t.Fatalf("expected C normal, got %q", c.Status)
	}
}
