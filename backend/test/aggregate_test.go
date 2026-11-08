package test

import (
	"testing"

	"harmony/infra/model/record"
)

// TestCrossBookAggregation 对应 docs/02「跨本子聚合」：
// 张三在升学宴中，人情礼本现金 500、火炮本火炮 100；人时间线这一组 inCent=60000，parts 拼出 500+100。
func TestCrossBookAggregation(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	renqingBook := h.createBook("人情礼本", "renqing", 0)
	huopaoBook := h.createBook("火炮本", "huopao", 0)

	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: renqingBook,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: huopaoBook,
		AmountCent: 10000, Direction: "in", OccurredOn: "2011-08-20",
	})

	timeline, err := h.viewSrv.PersonTimeline(h.ctx(), record.Query{PersonID: personID})
	requireOK(t, err)
	if len(timeline.Groups) != 1 {
		t.Fatalf("expected 1 timeline group, got %d", len(timeline.Groups))
	}
	group := timeline.Groups[0]
	if group.Event.Name != "升学宴" {
		t.Fatalf("expected event 升学宴, got %q", group.Event.Name)
	}
	if group.InCent != 60000 {
		t.Fatalf("expected inCent 60000, got %d", group.InCent)
	}
	parts := make(map[string]int64, len(group.Parts))
	var partsTotal int64
	for _, part := range group.Parts {
		parts[part.Label] = part.Cent
		partsTotal += part.Cent
	}
	if parts["现金"] != 50000 {
		t.Fatalf("expected 现金 part 50000, got %d (parts=%v)", parts["现金"], parts)
	}
	if parts["火炮"] != 10000 {
		t.Fatalf("expected 火炮 part 10000, got %d (parts=%v)", parts["火炮"], parts)
	}
	if partsTotal != group.InCent {
		t.Fatalf("expected parts sum %d to equal inCent %d", partsTotal, group.InCent)
	}
}

// TestConsistentCalibers 对应 docs/02「口径一致」：
// 同一批数据下人汇总、事项汇总、待还清单、年统计金额完全一致。
func TestConsistentCalibers(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	renqingBook := h.createBook("人情礼本", "renqing", 0)
	huopaoBook := h.createBook("火炮本", "huopao", 0)

	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: renqingBook,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: huopaoBook,
		AmountCent: 10000, Direction: "in", OccurredOn: "2011-08-20",
	})

	personDetail, err := h.personSrv.Detail(h.ctx(), personID)
	requireOK(t, err)
	eventSummary, err := h.viewSrv.EventSummary(h.ctx(), eventID)
	requireOK(t, err)
	pending, err := h.viewSrv.Pending(h.ctx(), "owed")
	requireOK(t, err)
	year, err := h.statsSrv.Year(h.ctx(), 2011)
	requireOK(t, err)

	const want = int64(60000)
	if personDetail.Summary.InCent != want {
		t.Fatalf("person summary inCent: want %d, got %d", want, personDetail.Summary.InCent)
	}
	if eventSummary.Summary.InCent != want {
		t.Fatalf("event summary inCent: want %d, got %d", want, eventSummary.Summary.InCent)
	}
	if pending.TotalCent != want {
		t.Fatalf("pending totalCent: want %d, got %d", want, pending.TotalCent)
	}
	if year.InCent != want {
		t.Fatalf("year inCent: want %d, got %d", want, year.InCent)
	}
	if personDetail.Summary.InCent != eventSummary.Summary.InCent ||
		eventSummary.Summary.InCent != pending.TotalCent ||
		pending.TotalCent != year.InCent {
		t.Fatalf("calibers disagree: person=%d event=%d pending=%d year=%d",
			personDetail.Summary.InCent, eventSummary.Summary.InCent, pending.TotalCent, year.InCent)
	}
}

// TestGiftValuation 对应 docs/02「礼品估价」：
// 现金 0、礼品「被面」、估价 200：total=20000，统计只计一次。
func TestGiftValuation(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	created := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 0, GiftName: "被面", GiftValueCent: 20000,
		Direction: "in", OccurredOn: "2011-08-20",
	})
	if created.TotalCent != 20000 {
		t.Fatalf("expected totalCent 20000, got %d", created.TotalCent)
	}
	if created.ItemKind != "gift" {
		t.Fatalf("expected itemKind gift, got %q", created.ItemKind)
	}
	if h.countRecords() != 1 {
		t.Fatalf("expected exactly 1 record, got %d", h.countRecords())
	}

	bookDetail, err := h.bookSrv.Detail(h.ctx(), bookID)
	requireOK(t, err)
	if bookDetail.InCent != 20000 {
		t.Fatalf("book inCent: want 20000, got %d", bookDetail.InCent)
	}
	year, err := h.statsSrv.Year(h.ctx(), 2011)
	requireOK(t, err)
	if year.InCent != 20000 {
		t.Fatalf("year inCent: want 20000 (counted once), got %d", year.InCent)
	}
	yearBook, err := h.statsSrv.Book(h.ctx(), bookID)
	requireOK(t, err)
	var giftCent int64
	for _, part := range yearBook.ByItem {
		if part.ItemKind == "gift" {
			giftCent = part.Cent
		}
	}
	if giftCent != 20000 {
		t.Fatalf("expected gift part 20000 once, got %d", giftCent)
	}
}
