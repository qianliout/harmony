package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	recordapi "harmony/component/record/api"
	statsapi "harmony/component/stats/api"
	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/record"
	"harmony/infra/model/stats"
)

// TestRecordPersonKeywordFilter 对应 docs/10 10.2「亲友文本筛选」：
// personKeyword 命中姓名包含、全拼前缀、首字母前缀，不命中拼音中段，并能与其他筛选组合。
func TestRecordPersonKeywordFilter(t *testing.T) {
	h := newHarness(t)
	eventID := h.createEvent("升学宴", "school", "2024-06-01", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)
	zhangID := h.createPerson("张三")
	liID := h.createPerson("李四")

	h.createRecord(record.CreateParam{
		PersonID: zhangID, EventID: eventID, BookID: bookID,
		AmountCent: 10000, Direction: "in", OccurredOn: "2024-06-01",
	})
	h.createRecord(record.CreateParam{
		PersonID: liID, EventID: eventID, BookID: bookID,
		AmountCent: 20000, Direction: "in", OccurredOn: "2024-06-01",
	})

	cases := []struct {
		name    string
		keyword string
		want    int64
	}{
		{"姓名包含", "张三", 1},
		{"全拼前缀", "zhang", 1},
		{"首字母前缀", "zs", 1},
		{"拼音中段不命中", "ang", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			items, total, err := h.recordSrv.List(h.ctx(), record.Query{PersonKeyword: c.keyword}, nil)
			requireOK(t, err)
			if total != c.want || int64(len(items)) != c.want {
				t.Fatalf("personKeyword %q: want %d, got total=%d items=%d", c.keyword, c.want, total, len(items))
			}
			for i := range items {
				if items[i].PersonID != zhangID {
					t.Fatalf("personKeyword %q matched person %d", c.keyword, items[i].PersonID)
				}
			}
		})
	}

	// 与其他筛选组合：命中姓名的同时限定送方向，应为空。
	_, total, err := h.recordSrv.List(h.ctx(), record.Query{PersonKeyword: "zhang", Direction: "out"}, nil)
	requireOK(t, err)
	if total != 0 {
		t.Fatalf("personKeyword + direction should be empty, got %d", total)
	}

	// 搜索接口复用同一查询，参数必须能透传。
	hits, total, err := h.searchSrv.Records(h.ctx(), record.Query{PersonKeyword: "zs"}, 1, 10)
	requireOK(t, err)
	if total != 1 || len(hits) != 1 || hits[0].PersonID != zhangID {
		t.Fatalf("search records personKeyword: total=%d hits=%d", total, len(hits))
	}
}

// TestRecordSummary 对应 docs/10 10.1「记录列表汇总」：
// summary 与列表使用同一筛选条件，条数与收、送、未回礼金额不受分页影响；空结果为 0。
func TestRecordSummary(t *testing.T) {
	h := newHarness(t)
	eventID := h.createEvent("升学宴", "school", "2024-06-01", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)
	otherBookID := h.createBook("另外一本", "renqing", 0)
	zhangID := h.createPerson("张三")
	liID := h.createPerson("李四")

	h.createRecord(record.CreateParam{
		PersonID: zhangID, EventID: eventID, BookID: bookID,
		AmountCent: 10000, Direction: "in", OccurredOn: "2024-01-05",
	})
	h.createRecord(record.CreateParam{
		PersonID: zhangID, EventID: eventID, BookID: bookID,
		AmountCent: 20000, Direction: "in", OccurredOn: "2024-01-06",
		Status: consts.RecordStatusRepaid, Confirm: true,
	})
	h.createRecord(record.CreateParam{
		PersonID: liID, EventID: eventID, BookID: otherBookID,
		AmountCent: 30000, Direction: "out", OccurredOn: "2024-02-05",
	})

	summary, err := h.recordSrv.Summary(h.ctx(), record.Query{})
	requireOK(t, err)
	want := record.Summary{Count: 3, InCent: 30000, OutCent: 30000, PendingCent: 10000}
	if summary != want {
		t.Fatalf("summary: want %+v, got %+v", want, summary)
	}

	// 分页只影响列表条数，不影响汇总。
	items, total, err := h.recordSrv.List(h.ctx(), record.Query{}, &modelfilter.Filter{Offset: 0, Limit: 2})
	requireOK(t, err)
	if total != 3 || len(items) != 2 {
		t.Fatalf("paged list: want total 3 items 2, got total=%d items=%d", total, len(items))
	}
	pagedSummary, err := h.recordSrv.Summary(h.ctx(), record.Query{})
	requireOK(t, err)
	if pagedSummary != summary {
		t.Fatalf("summary changed with paging: want %+v, got %+v", summary, pagedSummary)
	}

	// 组合筛选：只统计张三在人情礼本的记录。
	filtered, err := h.recordSrv.Summary(h.ctx(), record.Query{PersonID: zhangID, BookID: bookID})
	requireOK(t, err)
	wantFiltered := record.Summary{Count: 2, InCent: 30000, PendingCent: 10000}
	if filtered != wantFiltered {
		t.Fatalf("filtered summary: want %+v, got %+v", wantFiltered, filtered)
	}

	// 只选状态时按收礼处理，与列表口径一致。
	statusSummary, err := h.recordSrv.Summary(h.ctx(), record.Query{Status: consts.RecordStatusPending})
	requireOK(t, err)
	wantStatus := record.Summary{Count: 1, InCent: 10000, PendingCent: 10000}
	if statusSummary != wantStatus {
		t.Fatalf("status summary: want %+v, got %+v", wantStatus, statusSummary)
	}

	// 空结果汇总为 0。
	empty, err := h.recordSrv.Summary(h.ctx(), record.Query{Keyword: "没有这条记录"})
	requireOK(t, err)
	if empty != (record.Summary{}) {
		t.Fatalf("empty summary: want zeros, got %+v", empty)
	}
}

// TestPendingRecords 对应 docs/10 10.3「回礼清单明细」：
// 每个人的明细来自一次批量查询；收方向只含待还记录，送方向排除回礼生成的送礼记录。
func TestPendingRecords(t *testing.T) {
	h := newHarness(t)
	eventID := h.createEvent("升学宴", "school", "2024-06-01", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)
	personID := h.createPerson("张三")

	first := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 10000, Direction: "in", OccurredOn: "2024-01-01",
	})
	second := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 20000, Direction: "in", OccurredOn: "2024-02-01", Confirm: true,
	})
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 40000, Direction: "in", OccurredOn: "2024-03-01", Confirm: true,
	})

	owed, err := h.viewSrv.Pending(h.ctx(), "owed")
	requireOK(t, err)
	if len(owed.Persons) != 1 {
		t.Fatalf("owed persons: want 1, got %d", len(owed.Persons))
	}
	person := owed.Persons[0]
	if person.Count != 3 || len(person.Records) != 3 {
		t.Fatalf("owed records: want 3, got count=%d records=%d", person.Count, len(person.Records))
	}
	if owed.Truncated {
		t.Fatalf("owed should not be truncated")
	}

	// 还掉一笔后，明细同步减少，且金额与 count 一致。
	_, err = h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{first.ID},
		Event:         &record.EventParam{ID: eventID},
		OccurredOn:    "2024-04-01",
		DatePrecision: "day",
		Items:         []record.RepayItem{{BookID: bookID, AmountCent: 10000}},
	})
	requireOK(t, err)

	owed, err = h.viewSrv.Pending(h.ctx(), "owed")
	requireOK(t, err)
	person = owed.Persons[0]
	if person.Count != 2 || len(person.Records) != 2 {
		t.Fatalf("owed after repay: want 2, got count=%d records=%d", person.Count, len(person.Records))
	}
	gotIDs := map[uint64]bool{}
	for _, item := range person.Records {
		gotIDs[item.ID] = true
		if item.IsRepay {
			t.Fatalf("owed record %d should not be repay", item.ID)
		}
	}
	if !gotIDs[second.ID] {
		t.Fatalf("owed records missing %d: %+v", second.ID, person.Records)
	}

	// 送方向的待还明细：只包含真正的待还送礼记录。
	outRecord := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 25000, Direction: "out", OccurredOn: "2024-05-01", Confirm: true,
	})
	owing, err := h.viewSrv.Pending(h.ctx(), "owing")
	requireOK(t, err)
	if len(owing.Persons) != 1 || len(owing.Persons[0].Records) != 1 {
		t.Fatalf("owing: want 1 person 1 record, got %+v", owing.Persons)
	}
	if owing.Persons[0].Records[0].ID != outRecord.ID {
		t.Fatalf("owing record: want %d, got %d", outRecord.ID, owing.Persons[0].Records[0].ID)
	}

	// 把回礼生成的送礼记录状态改成待还，验证序列化后仍按 IsRepay 过滤。
	repayLinks, err := h.recordStore.LinksByOrigin(h.tx, 1, first.ID)
	requireOK(t, err)
	if len(repayLinks) != 1 {
		t.Fatalf("expected 1 repay link, got %d", len(repayLinks))
	}
	repayRecord, err := h.recordStore.Get(h.tx, 1, repayLinks[0].RepayRecordID)
	requireOK(t, err)
	repayRecord.Status = consts.RecordStatusPending
	requireOK(t, h.recordStore.Update(h.tx, repayRecord))

	owing, err = h.viewSrv.Pending(h.ctx(), "owing")
	requireOK(t, err)
	if len(owing.Persons) != 1 || len(owing.Persons[0].Records) != 1 {
		t.Fatalf("owing should exclude repay records, got %+v", owing.Persons)
	}
	for _, item := range owing.Persons[0].Records {
		if item.IsRepay {
			t.Fatalf("owing record %d should not be repay", item.ID)
		}
	}
}

// TestStatsYearSplitAndAll 对应 docs/10 10.4「统计」：
// 事项类型拆分收送、最近往来日期、year=all 与单年合计一致。
func TestStatsYearSplitAndAll(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	event2011 := h.createEvent("升学宴", "school", "2011-08-20", "day")
	event2024 := h.createEvent("婚礼", "wedding", "2024-05-01", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: event2011, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: event2011, BookID: bookID,
		AmountCent: 20000, Direction: "out", OccurredOn: "2011-09-01", Confirm: true,
	})
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: event2024, BookID: bookID,
		AmountCent: 10000, Direction: "in", OccurredOn: "2024-05-01",
	})

	year2011, err := h.statsSrv.Year(h.ctx(), 2011)
	requireOK(t, err)
	school := findEventTypeStat(t, year2011, "school")
	if school.Cent != 70000 || school.InCent != 50000 || school.OutCent != 20000 {
		t.Fatalf("2011 school split: want cent 70000 in 50000 out 20000, got %+v", school)
	}
	if school.RecordCount != 2 || school.EventCount != 1 {
		t.Fatalf("2011 school counts: got %+v", school)
	}
	if len(year2011.TopPersons) != 1 || year2011.TopPersons[0].LastDate != "2011-09-01" {
		t.Fatalf("2011 lastDate: got %+v", year2011.TopPersons)
	}

	year2024, err := h.statsSrv.Year(h.ctx(), 2024)
	requireOK(t, err)
	if year2024.InCent != 10000 || year2024.OutCent != 0 {
		t.Fatalf("2024 totals: got in=%d out=%d", year2024.InCent, year2024.OutCent)
	}

	all, err := h.statsSrv.Year(h.ctx(), -1)
	requireOK(t, err)
	if all.InCent != 60000 || all.OutCent != 20000 || all.RecordCount != 3 {
		t.Fatalf("all totals: got %+v", all)
	}
	if all.InCent != year2011.InCent+year2024.InCent || all.OutCent != year2011.OutCent+year2024.OutCent {
		t.Fatalf("all should equal sum of years: all=%+v 2011=%+v 2024=%+v", all, year2011, year2024)
	}
	if len(all.TopPersons) != 1 || all.TopPersons[0].LastDate != "2024-05-01" {
		t.Fatalf("all lastDate: got %+v", all.TopPersons)
	}

	// 月度仍是 12 个月合计，总和等于全年合计。
	if len(all.Monthly) != 12 {
		t.Fatalf("all monthly: want 12 months, got %d", len(all.Monthly))
	}
	var monthlyIn, monthlyOut int64
	for _, month := range all.Monthly {
		monthlyIn += month.InCent
		monthlyOut += month.OutCent
	}
	if monthlyIn != all.InCent || monthlyOut != all.OutCent {
		t.Fatalf("monthly sums: got in=%d out=%d, want in=%d out=%d", monthlyIn, monthlyOut, all.InCent, all.OutCent)
	}
	if all.Monthly[4].InCent != 10000 || all.Monthly[7].InCent != 50000 || all.Monthly[8].OutCent != 20000 {
		t.Fatalf("monthly buckets: got %+v", all.Monthly)
	}

	// year=0 保持“当前年”的既有行为。
	current, err := h.statsSrv.Year(h.ctx(), 0)
	requireOK(t, err)
	if current.Year != int64(time.Now().Year()) {
		t.Fatalf("year=0 should use current year, got %d", current.Year)
	}
}

// TestEventSummaryRepay 对应 docs/10 10.5「事项详情的回礼标记」：
// repayCount/repayCent 只统计回礼生成的送礼记录，普通送礼与回礼在明细里分开。
func TestEventSummaryRepay(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2024-06-01", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2024-06-01",
	})
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 12000, Direction: "out", OccurredOn: "2024-06-02", Confirm: true,
	})
	repayOrigin := h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2024-06-03", Confirm: true,
	})
	_, err := h.recordSrv.Repay(h.ctx(), record.RepayParam{
		OriginIDs:     []uint64{repayOrigin.ID},
		Event:         &record.EventParam{ID: eventID},
		OccurredOn:    "2024-06-04",
		DatePrecision: "day",
		Items:         []record.RepayItem{{BookID: bookID, AmountCent: 50000}},
	})
	requireOK(t, err)

	summary, err := h.viewSrv.EventSummary(h.ctx(), eventID)
	requireOK(t, err)
	if len(summary.Persons) != 1 {
		t.Fatalf("persons: want 1, got %d", len(summary.Persons))
	}
	person := summary.Persons[0]
	if person.RepayCount != 1 || person.RepayCent != 50000 {
		t.Fatalf("repay stats: want count 1 cent 50000, got %+v", person)
	}
	if person.OutCent != 62000 {
		t.Fatalf("outCent: want 62000, got %d", person.OutCent)
	}

	var repayLines, plainOutLines int
	for _, line := range person.Details {
		if line.Direction != consts.DirectionOut {
			continue
		}
		if line.IsRepay {
			repayLines++
			if line.Cent != 50000 {
				t.Fatalf("repay line cent: want 50000, got %d", line.Cent)
			}
			continue
		}
		plainOutLines++
		if line.Cent != 12000 {
			t.Fatalf("plain out line cent: want 12000, got %d", line.Cent)
		}
	}
	if repayLines != 1 || plainOutLines != 1 {
		t.Fatalf("out lines: want 1 repay + 1 plain, got repay=%d plain=%d (%+v)", repayLines, plainOutLines, person.Details)
	}
}

// findEventTypeStat 按事项类型取出统计行。
func findEventTypeStat(t *testing.T, year *stats.Year, eventType string) stats.EventTypeStat {
	t.Helper()
	for _, row := range year.ByEventType {
		if row.Type == eventType {
			return row
		}
	}
	t.Fatalf("event type %q not found in %+v", eventType, year.ByEventType)
	return stats.EventTypeStat{}
}

// newTestEngine 建立只绑定 owner 的最小路由，用于验证 API 层的参数解析与响应封装。
func newTestEngine(h *harness) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(h.ctx())
	})
	return engine
}

// TestRecordListAPIExposesSummary 验证 /records 读取 personKeyword 并返回 custom.summary。
func TestRecordListAPIExposesSummary(t *testing.T) {
	h := newHarness(t)
	eventID := h.createEvent("升学宴", "school", "2024-06-01", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)
	zhangID := h.createPerson("张三")
	h.createRecord(record.CreateParam{
		PersonID: zhangID, EventID: eventID, BookID: bookID,
		AmountCent: 12300, Direction: "in", OccurredOn: "2024-06-01",
	})

	engine := newTestEngine(h)
	recordapi.RegisterRoutes(engine.Group(""), recordapi.NewRecordAPI(h.recordSrv))

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/records?personKeyword=zhang", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Code   int64         `json:"code"`
		Data   []record.Resp `json:"data"`
		Total  int64         `json:"totalItems"`
		Custom struct {
			Summary record.Summary `json:"summary"`
		} `json:"custom"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v body=%s", err, recorder.Body.String())
	}
	if body.Code != http.StatusOK {
		t.Fatalf("code: want 200, got %d", body.Code)
	}
	if body.Total != 1 || len(body.Data) != 1 {
		t.Fatalf("personKeyword should filter to 1, got total=%d items=%d", body.Total, len(body.Data))
	}
	if body.Custom.Summary != (record.Summary{Count: 1, InCent: 12300, PendingCent: 12300}) {
		t.Fatalf("custom.summary mismatch: %+v", body.Custom.Summary)
	}
}

// TestStatsYearAPIAll 验证 /stats/year?year=all 不会被解析成当前年，而是全时间范围。
func TestStatsYearAPIAll(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)
	h.createRecord(record.CreateParam{
		PersonID: personID, EventID: eventID, BookID: bookID,
		AmountCent: 50000, Direction: "in", OccurredOn: "2011-08-20",
	})

	engine := newTestEngine(h)
	statsapi.RegisterRoutes(engine.Group(""), statsapi.NewStatsAPI(h.statsSrv))

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stats/year?year=all", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Code int64      `json:"code"`
		Data stats.Year `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v body=%s", err, recorder.Body.String())
	}
	if body.Data.Year != -1 {
		t.Fatalf("year=all should pass -1, got %d", body.Data.Year)
	}
	if body.Data.InCent != 50000 || body.Data.RecordCount != 1 {
		t.Fatalf("year=all totals: %+v", body.Data)
	}
}
