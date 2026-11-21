// Package test 是 harmony service 层的集成测试。
// 每个用例在独立事务里执行并回滚，TestMain 在临时目录建库并执行 schema 快照。
package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	bookservice "harmony/component/book/service"
	bookstore "harmony/component/book/store"
	"harmony/component/config"
	personservice "harmony/component/person/service"
	personstore "harmony/component/person/store"
	recognizeservice "harmony/component/recognize/service"
	recognizestore "harmony/component/recognize/store"
	recordservice "harmony/component/record/service"
	recordstore "harmony/component/record/store"
	"harmony/infra/consts"
	"harmony/infra/middleware"
	"harmony/infra/model/book"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/migrate"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "harmony-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	if err := setupTestDB(dir); err != nil {
		fmt.Fprintf(os.Stderr, "setup test database: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func setupTestDB(dir string) error {
	database, err := rdb.Open(filepath.Join(dir, "harmony-test.db"))
	if err != nil {
		return fmt.Errorf("open sqlite test database: %w", err)
	}
	db := database.Get()
	for _, statement := range migrate.Statements() {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("exec schema statement: %w", err)
		}
	}
	testDB = db
	return nil
}

// harness 为单个测试提供绑定事务的 DAO 与 service，用例结束后整体回滚。
type harness struct {
	t   *testing.T
	ctx context.Context

	personDao *personstore.PersonDao
	bookDao   *bookstore.BookDao
	recordDao *recordstore.RecordDao
	taskDao   *recognizestore.RecognizeTaskDao

	personSrv    *personservice.PersonSrv
	bookSrv      *bookservice.BookSrv
	recordSrv    *recordservice.RecordSrv
	recognizeSrv *recognizeservice.RecognizeSrv
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tx := testDB.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() {
		err := tx.Rollback().Error
		if err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
			t.Logf("rollback: %v", err)
		}
	})

	database := rdb.New(tx, "sqlite")
	h := &harness{
		t:         t,
		ctx:       middleware.WithOwner(context.Background(), consts.LocalOwnerID),
		personDao: personstore.NewPersonDao(database),
		bookDao:   bookstore.NewBookDao(database),
		recordDao: recordstore.NewRecordDao(database),
		taskDao:   recognizestore.NewRecognizeTaskDao(database),
	}
	h.personSrv = personservice.NewPersonSrv(h.personDao, h.recordDao)
	h.bookSrv = bookservice.NewBookSrv(h.bookDao, h.recordDao)
	h.recordSrv = recordservice.NewRecordSrv(h.recordDao, h.personDao, h.bookDao, h.personSrv)
	h.recognizeSrv = recognizeservice.NewRecognizeSrv(h.taskDao, h.personDao, h.bookDao, h.personSrv, h.recordSrv,
		nil, config.RecognizeConfig{TimeoutSeconds: 1}, t.TempDir())
	return h
}

func (h *harness) createPerson(name, remark string) int64 {
	h.t.Helper()
	param := &person.CreatePersonParam{OwnerID: consts.LocalOwnerID, Name: name, Remark: remark}
	view, err := h.personSrv.CreatePerson(h.ctx, param.Serialize())
	require.NoError(h.t, err)
	return view.ID
}

func (h *harness) createBook(name, category string) int64 {
	h.t.Helper()
	param := &book.CreateBookParam{OwnerID: consts.LocalOwnerID, Name: name, Category: category}
	view, err := h.bookSrv.CreateBook(h.ctx, param.Serialize())
	require.NoError(h.t, err)
	return view.ID
}

func (h *harness) createRecord(param *record.CreateRecordParam) *record.RecordView {
	h.t.Helper()
	param.OwnerID = consts.LocalOwnerID
	if param.OccurredOn == "" {
		param.OccurredOn = "2011-10-01"
	}
	param = param.Serialize()
	require.NoError(h.t, param.Check())
	view, err := h.recordSrv.CreateRecord(h.ctx, param)
	require.NoError(h.t, err)
	return view
}

func (h *harness) countRecords() int64 {
	h.t.Helper()
	_, total, err := h.recordDao.SearchRecord(h.ctx, &record.SearchRecordParam{OwnerID: consts.LocalOwnerID})
	require.NoError(h.t, err)
	return total
}

// requireCode 断言错误是 *response.ErrI18 且业务码一致；reason 非空时同时断言 custom.reason。
func requireCode(t *testing.T, err error, code int64, reason string) *response.ErrI18 {
	t.Helper()
	var errI18 *response.ErrI18
	require.ErrorAs(t, err, &errI18)
	require.Equal(t, code, errI18.Code, errI18.ZH)
	if reason != "" {
		require.Equal(t, reason, errI18.Custom["reason"])
	}
	return errI18
}
