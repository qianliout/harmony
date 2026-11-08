package test

import (
	"strings"
	"testing"

	recordservice "harmony/component/record/service"
	"harmony/infra/consts"
	"harmony/infra/model/recognize"
	"harmony/infra/model/record"
	"harmony/infra/utils"
)

// TestRecognizeCommitRollback 对应 docs/02「识别回滚」：
// 批量写入中有一项校验失败时整批回滚，committed 仍为 false。
func TestRecognizeCommitRollback(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("识别甲")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	task := &recognize.Task{
		OwnerID:     1,
		ImagePath:   "202601/rollback.jpg",
		ImageSHA256: strings.Repeat("a", 64),
		ImageSize:   10,
		Status:      consts.TaskStatusPending,
		Items:       "{}",
	}
	requireOK(t, h.recognizeStore.Create(h.tx, task))
	requireOK(t, h.recognizeStore.MarkSucceeded(h.tx, task.ID,
		`{"page":"1-10","pageTotalCent":107000,"sumCent":0,"items":[]}`, ""))

	before := h.countRecords()
	_, err := h.recognizeSrv.Commit(h.ctx(), task.ID, recognize.CommitParam{
		Items: []recognize.CommitItem{
			{
				PersonID: personID, EventID: eventID, BookID: bookID,
				AmountCent: 10000, OccurredOn: "2011-08-20", Direction: "in",
			},
			{
				PersonID: personID, EventID: eventID, BookID: 999999,
				AmountCent: 20000, OccurredOn: "2011-08-20", Direction: "in",
			},
		},
	})
	requireCode(t, err, 400)
	if after := h.countRecords(); after != before {
		t.Fatalf("expected rollback, record count %d -> %d", before, after)
	}
	detail, err := h.recognizeSrv.Detail(h.ctx(), task.ID)
	requireOK(t, err)
	if detail.Committed {
		t.Fatalf("expected task still uncommitted after rollback")
	}
}

// TestPaperRepay 对应 docs/02「纸本还礼」：
// 标注「11年国庆 孙女 100元 火炮20元」生成 2 条送礼共 12000 分、2 条关联；
// 原记录已还；事项 host_person_id 为该人；日期 2011-10-01、精度 month。
func TestPaperRepay(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张光轩")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	const note = "11年国庆 孙女 100元 火炮20元"
	origin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
		PaperNote: note,
	})

	parsed := recordservice.ParsePaperNoteText(note)
	if parsed.OnlyMark {
		t.Fatalf("expected repay drafts, got onlyMark")
	}
	if len(parsed.Repays) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(parsed.Repays))
	}
	draft := parsed.Repays[0]

	result, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{origin.ID},
		Event:         draft.Event,
		OccurredOn:    draft.OccurredOn,
		DatePrecision: draft.DatePrecision,
		PaperNote:     note,
		Items:         draft.Items,
	})
	requireOK(t, err)
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 repay records, got %d", len(result.Records))
	}
	if result.RepayCount != 2 {
		t.Fatalf("expected 2 links, got %d", result.RepayCount)
	}
	var sum int64
	for _, item := range result.Records {
		sum += item.TotalCent
	}
	if sum != 12000 {
		t.Fatalf("expected repay total 12000, got %d", sum)
	}
	if links := h.countRepayLinks(); links != 2 {
		t.Fatalf("expected 2 stored links, got %d", links)
	}

	repaid, err := h.recordStore.Get(h.tx, 1, origin.ID)
	requireOK(t, err)
	if repaid.Status != consts.RecordStatusRepaid {
		t.Fatalf("expected origin repaid, got %q", repaid.Status)
	}

	created, err := h.eventStore.Get(h.tx, 1, result.Records[0].EventID)
	requireOK(t, err)
	if created.HostPersonID == nil || *created.HostPersonID != personID {
		t.Fatalf("expected event host person %d, got %v", personID, created.HostPersonID)
	}
	if created.EventDate == nil || utils.DateString(*created.EventDate) != "2011-10-01" {
		t.Fatalf("expected event date 2011-10-01, got %v", created.EventDate)
	}
	if created.DatePrecision != consts.DatePrecisionMonth {
		t.Fatalf("expected month precision, got %q", created.DatePrecision)
	}
}

// TestRepayReusesEvent 对应 docs/02「事项复用」：
// 同一个人第二条标注也写「孙女」，复用已有事项，不新建。
func TestRepayReusesEvent(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张光轩")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	const note = "11年国庆 孙女 100元 火炮20元"
	draft := recordservice.ParsePaperNoteText(note).Repays[0]

	firstOrigin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20", PaperNote: note,
	})
	first, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs: []uint64{firstOrigin.ID}, Event: draft.Event,
		OccurredOn: draft.OccurredOn, DatePrecision: draft.DatePrecision, Items: draft.Items,
	})
	requireOK(t, err)
	reusedEventID := first.Records[0].EventID
	eventCount := h.countEvents()

	secondOrigin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-21",
		Confirm: true,
	})
	second, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs: []uint64{secondOrigin.ID}, Event: draft.Event,
		OccurredOn: draft.OccurredOn, DatePrecision: draft.DatePrecision, Items: draft.Items,
	})
	requireOK(t, err)
	if h.countEvents() != eventCount {
		t.Fatalf("expected event count %d to stay, got %d", eventCount, h.countEvents())
	}
	if second.Records[0].EventID != reusedEventID {
		t.Fatalf("expected reuse event %d, got %d", reusedEventID, second.Records[0].EventID)
	}
}

// TestOnlyMark 对应 docs/02「只有记号」：
// 标注「✓」解析 onlyMark=true；mark-repaid 只改状态和原文，不新增记录。
func TestOnlyMark(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张光轩")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	parsed := recordservice.ParsePaperNoteText("✓")
	if !parsed.OnlyMark {
		t.Fatalf("expected onlyMark=true")
	}
	if len(parsed.Repays) != 0 {
		t.Fatalf("expected no drafts, got %d", len(parsed.Repays))
	}

	origin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20", PaperNote: "✓",
	})
	before := h.countRecords()
	requireOK(t, h.recordSrv.MarkRepaid(h.ctx(), origin.ID, record.MarkRepaidParam{PaperNote: "✓ 已回"}))
	if after := h.countRecords(); after != before {
		t.Fatalf("mark-repaid must not create records: %d -> %d", before, after)
	}
	updated, err := h.recordStore.Get(h.tx, 1, origin.ID)
	requireOK(t, err)
	if updated.Status != consts.RecordStatusRepaid {
		t.Fatalf("expected repaid, got %q", updated.Status)
	}
	if updated.PaperNote != "✓ 已回" {
		t.Fatalf("expected paper note updated, got %q", updated.PaperNote)
	}
}

// TestOneNoteTwoRepays 对应 docs/02「一条两次」：
// 「10年8月1日 生日 500 / 11年正月 50」解析出 2 个草稿；第二个没有事项；
// 两次都关联同一条原记录；第二次归入占位事项。
func TestOneNoteTwoRepays(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张光轩")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	parsed := recordservice.ParsePaperNoteText("10年8月1日 生日 500 / 11年正月 50")
	if len(parsed.Repays) != 2 {
		t.Fatalf("expected 2 drafts, got %d", len(parsed.Repays))
	}
	first, second := parsed.Repays[0], parsed.Repays[1]
	if first.Event == nil || first.Event.Name != "生日" {
		t.Fatalf("first draft event: %+v", first.Event)
	}
	if first.OccurredOn != "2010-08-01" || first.DatePrecision != consts.DatePrecisionDay {
		t.Fatalf("first draft date: %s/%s", first.OccurredOn, first.DatePrecision)
	}
	if len(first.Items) != 1 || first.Items[0].AmountCent != 50000 {
		t.Fatalf("first draft items: %+v", first.Items)
	}
	if second.Event != nil {
		t.Fatalf("second draft should have no event, got %+v", second.Event)
	}
	if second.OccurredOn != "2011-02-01" || second.DatePrecision != consts.DatePrecisionMonth {
		t.Fatalf("second draft date: %s/%s", second.OccurredOn, second.DatePrecision)
	}
	if len(second.Items) != 1 || second.Items[0].AmountCent != 5000 {
		t.Fatalf("second draft items: %+v", second.Items)
	}

	origin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})

	_, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs: []uint64{origin.ID}, Event: first.Event,
		OccurredOn: first.OccurredOn, DatePrecision: first.DatePrecision,
		MarkRepaid: boolPtr(false), Items: first.Items,
	})
	requireOK(t, err)
	secondResult, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs: []uint64{origin.ID}, Event: second.Event,
		OccurredOn: second.OccurredOn, DatePrecision: second.DatePrecision,
		Items: second.Items,
	})
	requireOK(t, err)

	links, err := h.recordStore.LinksByOrigin(h.tx, 1, origin.ID)
	requireOK(t, err)
	if len(links) != 2 {
		t.Fatalf("expected 2 links to the same origin, got %d", len(links))
	}
	placeholder, err := h.eventStore.Placeholder(h.tx, 1)
	requireOK(t, err)
	if secondResult.Records[0].EventID != placeholder.ID {
		t.Fatalf("expected second repay in placeholder event %d, got %d", placeholder.ID, secondResult.Records[0].EventID)
	}
}

// TestDefaultEvent 对应 docs/02「默认事项」：
// 礼簿设置 default_event_id 后，CreateInTx 不传 eventId 时使用它。
func TestDefaultEvent(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	defaultEventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("默认事项本", "renqing", defaultEventID)

	created, err := h.recordSrv.CreateInTx(h.ctx(), h.tx, record.CreateParam{
		PersonID: personID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})
	requireOK(t, err)
	if created.EventID != defaultEventID {
		t.Fatalf("expected default event %d, got %d", defaultEventID, created.EventID)
	}
}
