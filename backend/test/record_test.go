package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/record"
)

func TestRecordCreate(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三", "")
	bookID := h.createBook("人情礼本", "")

	incoming := h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 50000, Matter: "忽略"})
	assert.Equal(t, bookID, incoming.BookID)
	assert.Equal(t, "人情礼本", incoming.BookName)
	assert.Equal(t, "张三", incoming.PersonDisplayName)
	assert.Empty(t, incoming.Matter, "收礼不记事情")

	outgoing := h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "out", AmountCent: 20000, Matter: "张三升学酒"})
	assert.Equal(t, int64(0), outgoing.BookID, "送礼不挂礼簿")
	assert.Equal(t, "张三升学酒", outgoing.Matter)
}

func TestRecordCreateByNameReusesOrCreatesPerson(t *testing.T) {
	h := newHarness(t)
	existingID := h.createPerson("张三", "")
	bookID := h.createBook("人情礼本", "")

	reused := h.createRecord(&record.CreateRecordParam{PersonName: "张三", BookID: bookID, Direction: "in", AmountCent: 100})
	assert.Equal(t, existingID, reused.PersonID)

	created := h.createRecord(&record.CreateRecordParam{PersonName: "王五", PersonRemark: "表叔", BookID: bookID, Direction: "in", AmountCent: 100})
	assert.NotEqual(t, existingID, created.PersonID)
	assert.Equal(t, "王五 · 表叔", created.PersonDisplayName)
}

func TestRecordCreateErrors(t *testing.T) {
	cases := []struct {
		name     string
		param    *record.CreateRecordParam
		wantCode int64
		wantText string
	}{
		{name: "incoming without book fails check", param: &record.CreateRecordParam{PersonName: "张三", Direction: "in"}, wantText: "收礼需要选择礼簿"},
		{name: "negative amount fails check", param: &record.CreateRecordParam{PersonName: "张三", Direction: "out", AmountCent: -1}, wantText: "金额不能为负数"},
		{name: "bad date fails check", param: &record.CreateRecordParam{PersonName: "张三", Direction: "out", OccurredOn: "2011/1/1"}, wantText: "日期格式"},
		{name: "unknown book rejected", param: &record.CreateRecordParam{PersonName: "张三", Direction: "in", BookID: 999999}, wantCode: 400},
		{name: "unknown person id rejected", param: &record.CreateRecordParam{PersonID: 999999, Direction: "out"}, wantCode: 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			param := tc.param
			param.OwnerID = consts.LocalOwnerID
			param = param.Serialize()
			if tc.wantText != "" {
				err := param.Check()
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantText)
				return
			}
			require.NoError(t, param.Check())
			_, err := h.recordSrv.CreateRecord(h.ctx, param)
			requireCode(t, err, tc.wantCode, "")
			assert.Equal(t, int64(0), h.countRecords())
		})
	}
}

func TestRecordSearchAndSummary(t *testing.T) {
	h := newHarness(t)
	personA := h.createPerson("张三", "")
	personB := h.createPerson("李四", "")
	bookA := h.createBook("人情礼本", "")
	bookB := h.createBook("火炮本", "huopao")
	h.createRecord(&record.CreateRecordParam{PersonID: personA, BookID: bookA, Direction: "in", AmountCent: 50000, OccurredOn: "2011-08-20"})
	h.createRecord(&record.CreateRecordParam{PersonID: personB, BookID: bookB, Direction: "in", AmountCent: 10000, OccurredOn: "2011-09-01"})
	h.createRecord(&record.CreateRecordParam{PersonID: personA, Direction: "out", AmountCent: 20000, Matter: "李四结婚", OccurredOn: "2011-10-01"})

	cases := []struct {
		name      string
		param     record.SearchRecordParam
		wantTotal int64
		wantIn    int64
		wantOut   int64
	}{
		{name: "ledger shows all", wantTotal: 3, wantIn: 60000, wantOut: 20000},
		{name: "book view only its incoming", param: record.SearchRecordParam{BookID: bookA}, wantTotal: 1, wantIn: 50000},
		{name: "person view both directions", param: record.SearchRecordParam{PersonID: personA}, wantTotal: 2, wantIn: 50000, wantOut: 20000},
		{name: "direction filter", param: record.SearchRecordParam{Direction: "out"}, wantTotal: 1, wantOut: 20000},
		{name: "date range is inclusive", param: record.SearchRecordParam{From: "2011-09-01", To: "2011-10-01"}, wantTotal: 2, wantIn: 10000, wantOut: 20000},
		{name: "keyword hits person name and matter", param: record.SearchRecordParam{Keyword: "李四"}, wantTotal: 2, wantIn: 10000, wantOut: 20000},
		{name: "keyword hits pinyin initials", param: record.SearchRecordParam{Keyword: "ls"}, wantTotal: 1, wantIn: 10000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			searchParam := tc.param
			searchParam.OwnerID = consts.LocalOwnerID
			searchParam.Filter = &modelfilter.Filter{Limit: 20}
			items, total, err := h.recordSrv.SearchRecord(h.ctx, &searchParam)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.Len(t, items, int(tc.wantTotal))

			statParam := &record.StatRecordParam{
				OwnerID:   consts.LocalOwnerID,
				PersonID:  tc.param.PersonID,
				BookID:    tc.param.BookID,
				Direction: tc.param.Direction,
				From:      tc.param.From,
				To:        tc.param.To,
				Keyword:   tc.param.Keyword,
			}
			summary, err := h.recordSrv.SummaryRecord(h.ctx, statParam)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, summary.Count)
			assert.Equal(t, tc.wantIn, summary.InCent)
			assert.Equal(t, tc.wantOut, summary.OutCent)
		})
	}
}

func TestRecordSearchOrdersByDateDesc(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三", "")
	h.createRecord(&record.CreateRecordParam{PersonID: personID, Direction: "out", OccurredOn: "2011-01-01"})
	h.createRecord(&record.CreateRecordParam{PersonID: personID, Direction: "out", OccurredOn: "2012-01-01"})

	items, _, err := h.recordSrv.SearchRecord(h.ctx, &record.SearchRecordParam{OwnerID: consts.LocalOwnerID})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "2012-01-01", items[0].OccurredOn)
}

func TestRecordUpdateAndDelete(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三", "")
	bookID := h.createBook("人情礼本", "")
	created := h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 50000})

	updateParam := &record.UpdateRecordParam{
		ID: created.ID, OwnerID: consts.LocalOwnerID, PersonID: personID, BookID: bookID,
		Direction: "out", AmountCent: 60000, Matter: "回礼", OccurredOn: "2011-08-20",
	}
	updateParam = updateParam.Serialize()
	require.NoError(t, updateParam.Check())
	updated, err := h.recordSrv.UpdateRecord(h.ctx, updateParam)
	require.NoError(t, err)
	assert.Equal(t, int64(60000), updated.AmountCent)
	assert.Equal(t, "out", updated.Direction)
	assert.Equal(t, int64(0), updated.BookID, "改成送礼后清空礼簿")
	assert.Equal(t, "回礼", updated.Matter)

	require.NoError(t, h.recordSrv.DeleteRecord(h.ctx, &record.GetRecordParam{ID: created.ID, OwnerID: consts.LocalOwnerID}))
	assert.Equal(t, int64(0), h.countRecords())
	err = h.recordSrv.DeleteRecord(h.ctx, &record.GetRecordParam{ID: created.ID, OwnerID: consts.LocalOwnerID})
	requireCode(t, err, 404, "")
}
