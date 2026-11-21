package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harmony/infra/consts"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
)

func TestPersonCRUD(t *testing.T) {
	h := newHarness(t)

	personID := h.createPerson("张三", "")
	view, err := h.personSrv.GetPerson(h.ctx, &person.GetPersonParam{ID: personID, OwnerID: consts.LocalOwnerID})
	require.NoError(t, err)
	assert.Equal(t, "张三", view.DisplayName)

	items, total, err := h.personSrv.SearchPerson(h.ctx, &person.SearchPersonParam{OwnerID: consts.LocalOwnerID, Keyword: "zs"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, personID, items[0].ID)

	updateParam := &person.UpdatePersonParam{ID: personID, OwnerID: consts.LocalOwnerID, Name: "张三", Remark: "村东"}
	updated, err := h.personSrv.UpdatePerson(h.ctx, updateParam)
	require.NoError(t, err)
	assert.Equal(t, "张三 · 村东", updated.DisplayName)

	require.NoError(t, h.personSrv.DeletePerson(h.ctx, &person.GetPersonParam{ID: personID, OwnerID: consts.LocalOwnerID}))
	_, err = h.personSrv.GetPerson(h.ctx, &person.GetPersonParam{ID: personID, OwnerID: consts.LocalOwnerID})
	requireCode(t, err, 404, "")
}

func TestPersonDuplicateNeedsRemark(t *testing.T) {
	cases := []struct {
		name    string
		remark  string
		wantErr bool
	}{
		{name: "same name without remark is rejected", remark: "", wantErr: true},
		{name: "same name with remark is allowed", remark: "村西"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.createPerson("张三", "")

			param := &person.CreatePersonParam{OwnerID: consts.LocalOwnerID, Name: "张三", Remark: tc.remark}
			view, err := h.personSrv.CreatePerson(h.ctx, param)
			if tc.wantErr {
				requireCode(t, err, 409, consts.ReasonDuplicatePerson)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "张三 · 村西", view.DisplayName)
		})
	}
}

func TestPersonRenameToDuplicateRejected(t *testing.T) {
	h := newHarness(t)
	h.createPerson("张三", "")
	otherID := h.createPerson("李四", "")

	param := &person.UpdatePersonParam{ID: otherID, OwnerID: consts.LocalOwnerID, Name: "张三"}
	_, err := h.personSrv.UpdatePerson(h.ctx, param)
	requireCode(t, err, 409, consts.ReasonDuplicatePerson)
}

func TestPersonDeleteWithRecords(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三", "")
	bookID := h.createBook("人情礼本", "")
	h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 10000})

	err := h.personSrv.DeletePerson(h.ctx, &person.GetPersonParam{ID: personID, OwnerID: consts.LocalOwnerID})
	requireCode(t, err, 409, consts.ReasonHasReference)
}

func TestPersonViewCarriesSummary(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三", "")
	bookID := h.createBook("人情礼本", "")
	h.createRecord(&record.CreateRecordParam{PersonID: personID, BookID: bookID, Direction: "in", AmountCent: 50000, OccurredOn: "2011-08-20"})
	h.createRecord(&record.CreateRecordParam{PersonID: personID, Direction: "out", AmountCent: 20000, OccurredOn: "2012-01-02"})

	view, err := h.personSrv.GetPerson(h.ctx, &person.GetPersonParam{ID: personID, OwnerID: consts.LocalOwnerID})
	require.NoError(t, err)
	assert.Equal(t, int64(50000), view.InCent)
	assert.Equal(t, int64(20000), view.OutCent)
	assert.Equal(t, int64(2), view.RecordCount)
	assert.Equal(t, "2012-01-02", view.LastOn)
}

func TestMatchPerson(t *testing.T) {
	cases := []struct {
		name       string
		seed       [][2]string
		matchName  string
		remark     string
		wantName   string
		wantNil    bool
		wantReason string
	}{
		{name: "no person returns nil", matchName: "张三", wantNil: true},
		{name: "single person with remark matches by name", seed: [][2]string{{"张三", "村东"}}, matchName: "张三", wantName: "张三 · 村东"},
		{name: "plain name preferred over remarked ones", seed: [][2]string{{"张三", "村东"}, {"张三", ""}}, matchName: "张三", wantName: "张三"},
		{name: "several remarked persons are ambiguous", seed: [][2]string{{"张三", "村东"}, {"张三", "村西"}}, matchName: "张三", wantReason: consts.ReasonPersonAmbiguous},
		{name: "explicit remark not found returns nil", seed: [][2]string{{"张三", "村东"}}, matchName: "张三", remark: "村西", wantNil: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			for _, seed := range tc.seed {
				h.createPerson(seed[0], seed[1])
			}
			param := &person.MatchPersonParam{OwnerID: consts.LocalOwnerID, PersonName: tc.matchName, PersonRemark: tc.remark}
			matched, err := h.personSrv.MatchPerson(h.ctx, param)
			if tc.wantReason != "" {
				requireCode(t, err, 409, tc.wantReason)
				return
			}
			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, matched)
				return
			}
			require.NotNil(t, matched)
			assert.Equal(t, tc.wantName, matched.DisplayName())
		})
	}
}
