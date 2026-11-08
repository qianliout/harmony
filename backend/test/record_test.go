package test

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"harmony/infra/consts"
	"harmony/infra/model/record"
)

func boolPtr(value bool) *bool { return &value }

// TestDuplicateConfirmation 对应 docs/02「重复提示」：
// 同人、同事项、同本子、同名目第二次提交 409 need_confirm；带 confirm 成功；换本子直接成功。
func TestDuplicateConfirmation(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)
	otherBookID := h.createBook("另一本人情礼", "renqing", 0)

	param := record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	}
	h.createRecord(param)

	_, err := h.recordSrv.Create(h.ctx(), param)
	errI18 := requireReason(t, err, consts.ReasonNeedConfirm)
	payload := confirmPayload(t, errI18)
	if payload.Kind != consts.ConfirmDuplicateRecord {
		t.Fatalf("expected kind %q, got %q", consts.ConfirmDuplicateRecord, payload.Kind)
	}
	if len(payload.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(payload.Conflicts))
	}

	confirmed := param
	confirmed.Confirm = true
	duplicated := h.createRecord(confirmed)
	if duplicated.BookID != bookID {
		t.Fatalf("expected confirmed record in book %d, got %d", bookID, duplicated.BookID)
	}

	otherBook := param
	otherBook.BookID = otherBookID
	direct := h.createRecord(otherBook)
	if direct.BookID != otherBookID {
		t.Fatalf("expected direct record in book %d, got %d", otherBookID, direct.BookID)
	}
	if h.countRecords() != 3 {
		t.Fatalf("expected 3 records, got %d", h.countRecords())
	}
}

// TestRepayManyToOne 对应 docs/02「还礼多对一」：
// 两笔收礼一次还清 → 1 条送礼记录、2 条关联，两笔收礼都变成已还。
func TestRepayManyToOne(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookA := h.createBook("人情礼 A", "renqing", 0)
	bookB := h.createBook("人情礼 B", "renqing", 0)

	first := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookA,
		AmountCent: 30000, Direction: "in", OccurredOn: "2011-08-20",
	})
	second := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookB,
		AmountCent: 30000, Direction: "in", OccurredOn: "2011-08-20",
	})

	result, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{first.ID, second.ID},
		OccurredOn:    "2011-09-01",
		DatePrecision: "day",
		Items:         []record.RepayItem{{AmountCent: 60000}},
	})
	requireOK(t, err)
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 repay record, got %d", len(result.Records))
	}
	if result.RepayCount != 2 {
		t.Fatalf("expected 2 links, got %d", result.RepayCount)
	}
	if total := h.countRepayLinks(); total != 2 {
		t.Fatalf("expected 2 stored links, got %d", total)
	}
	for _, originID := range []uint64{first.ID, second.ID} {
		links, err := h.recordStore.LinksByOrigin(h.tx, 1, originID)
		requireOK(t, err)
		if len(links) != 1 {
			t.Fatalf("origin %d: expected 1 link, got %d", originID, len(links))
		}
		origin, err := h.recordStore.Get(h.tx, 1, originID)
		requireOK(t, err)
		if origin.Status != consts.RecordStatusRepaid {
			t.Fatalf("origin %d: expected repaid, got %q", originID, origin.Status)
		}
	}
	if result.Records[0].Direction != consts.DirectionOut {
		t.Fatalf("expected repay direction out, got %q", result.Records[0].Direction)
	}
}

// TestRepayOneToMany 对应 docs/02「还礼一对多」：
// 一笔收礼分两次还，第一次 markRepaid=false，第二次 true。
func TestRepayOneToMany(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	origin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})

	first, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{origin.ID},
		MarkRepaid:    boolPtr(false),
		OccurredOn:    "2011-09-01",
		DatePrecision: "day",
		Items:         []record.RepayItem{{AmountCent: 20000}},
	})
	requireOK(t, err)
	if len(first.Records) != 1 || first.RepayCount != 1 {
		t.Fatalf("first repay: records=%d links=%d", len(first.Records), first.RepayCount)
	}
	stillPending, err := h.recordStore.Get(h.tx, 1, origin.ID)
	requireOK(t, err)
	if stillPending.Status != consts.RecordStatusPending {
		t.Fatalf("after markRepaid=false: expected pending, got %q", stillPending.Status)
	}

	second, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{origin.ID},
		MarkRepaid:    boolPtr(true),
		OccurredOn:    "2011-10-01",
		DatePrecision: "day",
		Items:         []record.RepayItem{{AmountCent: 30000}},
	})
	requireOK(t, err)
	if len(second.Records) != 1 || second.RepayCount != 1 {
		t.Fatalf("second repay: records=%d links=%d", len(second.Records), second.RepayCount)
	}
	repaid, err := h.recordStore.Get(h.tx, 1, origin.ID)
	requireOK(t, err)
	if repaid.Status != consts.RecordStatusRepaid {
		t.Fatalf("after markRepaid=true: expected repaid, got %q", repaid.Status)
	}
	if total := h.countRepayLinks(); total != 2 {
		t.Fatalf("expected 2 links across two repays, got %d", total)
	}
}

// TestDeleteProtection 对应 docs/02「删除保护」：
// 删除被关联的收礼 409 has_reference；删除还礼记录需确认，确认后收礼恢复待还。
func TestDeleteProtection(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	origin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})
	result, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{origin.ID},
		OccurredOn:    "2011-09-01",
		DatePrecision: "day",
		Items:         []record.RepayItem{{AmountCent: 50000}},
	})
	requireOK(t, err)
	repayID := result.Records[0].ID

	err = h.recordSrv.Delete(h.ctx(), origin.ID, false)
	errI18 := requireReason(t, err, consts.ReasonHasReference)
	if refCount := hasReferenceCount(t, errI18); refCount != 1 {
		t.Fatalf("expected refCount 1, got %d", refCount)
	}
	stillThere, getErr := h.recordStore.Get(h.tx, 1, origin.ID)
	if getErr != nil {
		t.Fatalf("origin should still exist: %v", getErr)
	}
	if stillThere.Status != consts.RecordStatusRepaid {
		t.Fatalf("expected origin still repaid, got %q", stillThere.Status)
	}

	err = h.recordSrv.Delete(h.ctx(), repayID, false)
	errI18 = requireReason(t, err, consts.ReasonNeedConfirm)
	payload := confirmPayload(t, errI18)
	if payload.Kind != consts.ConfirmUnlinkOnDelete {
		t.Fatalf("expected kind %q, got %q", consts.ConfirmUnlinkOnDelete, payload.Kind)
	}

	requireOK(t, h.recordSrv.Delete(h.ctx(), repayID, true))
	if links := h.countRepayLinks(); links != 0 {
		t.Fatalf("expected links removed, got %d", links)
	}
	restored, err := h.recordStore.Get(h.tx, 1, origin.ID)
	requireOK(t, err)
	if restored.Status != consts.RecordStatusPending {
		t.Fatalf("expected origin restored to pending, got %q", restored.Status)
	}
	_, err = h.recordStore.Get(h.tx, 1, repayID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected repay record deleted, got %v", err)
	}
}

func TestStatusLabelsAndFilters(t *testing.T) {
	h := newHarness(t)
	p := h.createPerson("标签人")
	e := h.createEvent("标签事", "wedding", "2024-02-02", "day")
	b := h.createBook("标签本", "renqing", 0)
	in, err := h.recordSrv.Create(h.ctx(), record.CreateParam{
		PersonID: p, EventID: e, BookID: b, Direction: "in",
		AmountCent: 10000, OccurredOn: "2024-02-02", DatePrecision: "day", Status: "pending",
	})
	requireOK(t, err)
	if in.StatusLabel != "未回礼" || in.IsRepay {
		t.Fatalf("unexpected in label %+v", in)
	}
	out, err := h.recordSrv.Create(h.ctx(), record.CreateParam{
		PersonID: p, EventID: e, BookID: b, Direction: "out",
		AmountCent: 10000, OccurredOn: "2024-02-02", DatePrecision: "day", Status: "pending",
		Confirm: true,
	})
	requireOK(t, err)
	if out.StatusLabel != "对方未回" {
		t.Fatalf("unexpected out label %q", out.StatusLabel)
	}
	repaid, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:  []uint64{in.ID},
		OccurredOn: "2024-03-03",
		Items:      []record.RepayItem{{BookID: b, ItemKind: "cash", AmountCent: 10000}},
	})
	requireOK(t, err)
	if len(repaid.Records) != 1 || repaid.Records[0].StatusLabel != "回礼" || !repaid.Records[0].IsRepay {
		t.Fatalf("unexpected repay record %+v", repaid.Records)
	}

	// 只传 status 时按收礼处理，不能混入送礼记录。
	list, total, err := h.recordSrv.List(h.ctx(), record.Query{Status: "repaid"}, nil)
	requireOK(t, err)
	if total != 1 || len(list) != 1 || list[0].Direction != "in" {
		t.Fatalf("status filter should imply direction=in, got total=%d %+v", total, list)
	}

	// 主维度筛选：类别与事项类型。
	byCategory, total, err := h.recordSrv.List(h.ctx(), record.Query{BookCategory: "renqing"}, nil)
	requireOK(t, err)
	if total != 3 || len(byCategory) != 3 {
		t.Fatalf("category filter got %d records", total)
	}
	_, total, err = h.recordSrv.List(h.ctx(), record.Query{EventType: "funeral"}, nil)
	requireOK(t, err)
	if total != 0 {
		t.Fatalf("event type filter should be empty, got %d", total)
	}
}

func TestSQLiteDateBoundary(t *testing.T) {
	h := newHarness(t)
	p := h.createPerson("边界人")
	e := h.createEvent("边界事", "other", "2011-10-31", "day")
	b := h.createBook("边界本", "renqing", 0)
	_, err := h.recordSrv.Create(h.ctx(), record.CreateParam{
		PersonID: p, EventID: e, BookID: b, Direction: "in",
		AmountCent: 5000, OccurredOn: "2011-10-31", DatePrecision: "day", Status: "pending",
	})
	requireOK(t, err)

	items, total, err := h.recordSrv.List(h.ctx(), record.Query{To: "2011-10-31"}, nil)
	requireOK(t, err)
	if total != 1 || len(items) != 1 {
		t.Fatalf("to=2011-10-31 should include the record, total=%d", total)
	}
	_, total, err = h.recordSrv.List(h.ctx(), record.Query{From: "2011-11-01"}, nil)
	requireOK(t, err)
	if total != 0 {
		t.Fatalf("from=2011-11-01 should exclude the record, total=%d", total)
	}
}
