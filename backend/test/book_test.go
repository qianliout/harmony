package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harmony/infra/consts"
	"harmony/infra/model/book"
	"harmony/infra/model/record"
)

func TestBookCRUD(t *testing.T) {
	h := newHarness(t)

	param := &book.CreateBookParam{OwnerID: consts.LocalOwnerID, Name: "人情礼本", Description: "2024 年", OccurredOn: "2024-10-01"}
	created, err := h.bookSrv.CreateBook(h.ctx, param.Serialize())
	require.NoError(t, err)
	assert.Equal(t, consts.BookCategoryRenqing, created.Category)
	assert.Equal(t, "人情礼", created.CategoryLabel)
	assert.Equal(t, "2024-10-01", created.OccurredOn)

	items, total, err := h.bookSrv.SearchBook(h.ctx, &book.SearchBookParam{OwnerID: consts.LocalOwnerID, Keyword: "人情"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	updateParam := &book.UpdateBookParam{ID: created.ID, OwnerID: consts.LocalOwnerID, Name: "火炮本", Category: "huopao", OccurredOn: "2025-02-03"}
	updated, err := h.bookSrv.UpdateBook(h.ctx, updateParam)
	require.NoError(t, err)
	assert.Equal(t, "火炮本", updated.Name)
	assert.Equal(t, "huopao", updated.Category)
	assert.Equal(t, "2025-02-03", updated.OccurredOn)
	assert.Empty(t, updated.Description)

	require.NoError(t, h.bookSrv.DeleteBook(h.ctx, &book.GetBookParam{ID: created.ID, OwnerID: consts.LocalOwnerID}))
	_, err = h.bookSrv.GetBook(h.ctx, &book.GetBookParam{ID: created.ID, OwnerID: consts.LocalOwnerID})
	requireCode(t, err, 404, "")
}

func TestBookCustomCategoryAndSummary(t *testing.T) {
	h := newHarness(t)
	bookID := h.createBook("升学酒", "升学")
	personID := h.createPerson("张三", "")
	h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 30000})
	h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 20000})

	view, err := h.bookSrv.GetBook(h.ctx, &book.GetBookParam{ID: bookID, OwnerID: consts.LocalOwnerID})
	require.NoError(t, err)
	assert.Equal(t, "升学", view.CategoryLabel)
	assert.Equal(t, int64(2), view.RecordCount)
	assert.Equal(t, int64(50000), view.InCent)
}

func TestBookDeleteWithRecords(t *testing.T) {
	h := newHarness(t)
	bookID := h.createBook("人情礼本", "")
	personID := h.createPerson("张三", "")
	h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 10000})

	err := h.bookSrv.DeleteBook(h.ctx, &book.GetBookParam{ID: bookID, OwnerID: consts.LocalOwnerID})
	requireCode(t, err, 409, consts.ReasonHasReference)
}

func TestBookParamCheck(t *testing.T) {
	cases := []struct {
		name    string
		param   *book.CreateBookParam
		wantErr string
	}{
		{name: "valid book passes", param: &book.CreateBookParam{OwnerID: 1, Name: " 人情礼本 "}},
		{name: "blank name rejected", param: &book.CreateBookParam{OwnerID: 1, Name: "  "}, wantErr: "礼簿名称不能为空"},
		{name: "invalid date rejected", param: &book.CreateBookParam{OwnerID: 1, Name: "人情礼本", OccurredOn: "2024/10/01"}, wantErr: "日期格式"},
		{name: "missing owner rejected", param: &book.CreateBookParam{Name: "人情礼本"}, wantErr: "ownerID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			param := tc.param.Serialize()
			err := param.Check()
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "人情礼本", param.Name)
		})
	}
}
