package test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harmony/infra/consts"
	"harmony/infra/model/recognize"
)

// seedSucceededTask 直接写入一个已识别完成的任务，绕开后台识别流程。
func (h *harness) seedSucceededTask(sha string) int64 {
	h.t.Helper()
	task := &recognize.RecognizeTask{
		OwnerID:     consts.LocalOwnerID,
		ImagePath:   "202601/" + sha + ".jpg",
		ImageSHA256: strings.Repeat(sha, 64),
		ImageSize:   10,
	}
	require.NoError(h.t, h.taskDao.CreateRecognizeTask(h.ctx, task))
	task.Status = consts.TaskStatusSucceeded
	task.Items = []recognize.RecognizeItem{{Seq: 1, Name: "张三", AmountCent: 50000}}
	task = task.Serialize()
	require.NoError(h.t, task.Check())
	require.NoError(h.t, h.taskDao.UpdateRecognizeTask(h.ctx, task.ID, task.ToUpdater()))
	return task.ID
}

func TestRecognizeCommit(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三", "")
	bookID := h.createBook("人情礼本", "")
	taskID := h.seedSucceededTask("a")

	param := &recognize.CommitRecognizeTaskParam{
		ID: taskID, OwnerID: consts.LocalOwnerID, BookID: bookID, OccurredOn: "2011-08-20",
		Items: []recognize.CommitRecognizeItem{
			{PersonID: personID, AmountCent: 50000},
			{PersonName: "王五", AmountCent: 10000},
			{PersonName: "王五", AmountCent: 20000},
		},
	}
	param = param.Serialize()
	require.NoError(t, param.Check())
	result, err := h.recognizeSrv.CommitRecognizeTask(h.ctx, param)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Count)
	assert.Equal(t, int64(80000), result.SumCent)
	assert.Equal(t, int64(1), result.NewPersons, "同批重复的新姓名只建一次")
	assert.Len(t, result.RecordIDs, 3)

	task, err := h.recognizeSrv.GetRecognizeTask(h.ctx, &recognize.GetRecognizeTaskParam{ID: taskID, OwnerID: consts.LocalOwnerID})
	require.NoError(t, err)
	assert.Equal(t, consts.TrueString, task.Committed)
	assert.Equal(t, int64(50000), task.SumCent)
	require.Len(t, task.Items, 1)

	_, err = h.recognizeSrv.CommitRecognizeTask(h.ctx, param)
	requireCode(t, err, 409, consts.ReasonStateConflict)
}

func TestRecognizeCommitValidatesBeforeWriting(t *testing.T) {
	cases := []struct {
		name     string
		bookID   int64
		seed     [][2]string
		items    []recognize.CommitRecognizeItem
		wantCode int64
	}{
		{name: "unknown book writes nothing", bookID: 999999, items: []recognize.CommitRecognizeItem{{PersonName: "张三", AmountCent: 100}}, wantCode: 400},
		{
			name:     "ambiguous name on later line writes nothing",
			seed:     [][2]string{{"张三", "村东"}, {"张三", "村西"}},
			items:    []recognize.CommitRecognizeItem{{PersonName: "李四", AmountCent: 100}, {PersonName: "张三", AmountCent: 100}},
			wantCode: 409,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			for _, seed := range tc.seed {
				h.createPerson(seed[0], seed[1])
			}
			bookID := tc.bookID
			if bookID == 0 {
				bookID = h.createBook("人情礼本", "")
			}
			taskID := h.seedSucceededTask("b")
			param := &recognize.CommitRecognizeTaskParam{ID: taskID, OwnerID: consts.LocalOwnerID, BookID: bookID, Items: tc.items}
			param = param.Serialize()
			require.NoError(t, param.Check())

			_, err := h.recognizeSrv.CommitRecognizeTask(h.ctx, param)
			errI18 := requireCode(t, err, tc.wantCode, "")
			assert.Equal(t, int64(0), h.countRecords())
			if tc.wantCode == 409 {
				assert.Contains(t, errI18.ZH, "第 2 行")
			}
		})
	}
}

func TestRecognizeCreateRequiresAPIKey(t *testing.T) {
	h := newHarness(t)
	param := &recognize.CreateRecognizeTaskParam{OwnerID: consts.LocalOwnerID, ImageData: []byte("\xff\xd8\xff\xe0jpeg")}
	param = param.Serialize()
	require.NoError(t, param.Check())

	_, err := h.recognizeSrv.CreateRecognizeTask(h.ctx, param)
	requireCode(t, err, 502, "")
}

func TestRecognizeCreateParamCheck(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{name: "jpeg accepted", data: []byte("\xff\xd8\xff\xe0jpeg")},
		{name: "empty image rejected", data: nil, wantErr: "请选择"},
		{name: "text file rejected", data: []byte("hello"), wantErr: "只支持"},
		{name: "oversize image rejected", data: append([]byte("\xff\xd8\xff\xe0"), make([]byte, consts.RecognizeImageMaxSize)...), wantErr: "5MB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			param := &recognize.CreateRecognizeTaskParam{OwnerID: consts.LocalOwnerID, ImageData: tc.data}
			param = param.Serialize()
			err := param.Check()
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, consts.MIMEJPEG, param.ImageMIME)
			assert.Len(t, param.ImageSHA256, 64)
		})
	}
}
