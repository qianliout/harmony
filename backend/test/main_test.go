// Package test 是 harmony service 层的集成测试。
// 每个用例在独立事务里执行并回滚，TestMain 重建 harmony_test 库并执行 schema 快照。
package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	authservice "harmony/component/auth/service"
	authstore "harmony/component/auth/store"
	bookservice "harmony/component/book/service"
	bookstore "harmony/component/book/store"
	"harmony/component/config"
	dictservice "harmony/component/dict/service"
	dictstore "harmony/component/dict/store"
	eventservice "harmony/component/event/service"
	eventstore "harmony/component/event/store"
	personservice "harmony/component/person/service"
	personstore "harmony/component/person/store"
	recognizeservice "harmony/component/recognize/service"
	recognizestore "harmony/component/recognize/store"
	recordservice "harmony/component/record/service"
	recordstore "harmony/component/record/store"
	searchservice "harmony/component/search/service"
	searchstore "harmony/component/search/store"
	settingservice "harmony/component/setting/service"
	settingstore "harmony/component/setting/store"
	statsservice "harmony/component/stats/service"
	statsstore "harmony/component/stats/store"
	viewservice "harmony/component/view/service"
	viewstore "harmony/component/view/store"
	"harmony/infra/model/book"
	"harmony/infra/model/event"
	"harmony/infra/model/person"
	"harmony/infra/middleware"
	"harmony/infra/model/record"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/migrate"
)

// testDB 是包级连接；每个用例从它开启事务，互不污染。
var (
	testDB  *gorm.DB
	tempDir string
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "harmony-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	tempDir = dir
	if err := setupTestDB(); err != nil {
		fmt.Fprintf(os.Stderr, "setup test database: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func setupTestDB() error {
	database, err := rdb.Open(filepath.Join(tempDir, "harmony-test.db"))
	if err != nil {
		return fmt.Errorf("open sqlite test database: %w", err)
	}
	db := database.Get()
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, statement := range migrate.Statements() {
			if err := tx.Exec(statement).Error; err != nil {
				snippet := statement
				if len(snippet) > 80 {
					snippet = snippet[:80]
				}
				return fmt.Errorf("exec %q: %w", snippet, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	testDB = db
	return nil
}

// harness 为单个测试提供绑定事务的 database 与全部 store/service。
type harness struct {
	t  *testing.T
	tx *gorm.DB

	dictStore      *dictstore.Store
	personStore    *personstore.Store
	bookStore      *bookstore.Store
	eventStore     *eventstore.Store
	recordStore    *recordstore.Store
	viewStore      *viewstore.Store
	statsStore     *statsstore.Store
	searchStore    *searchstore.Store
	settingStore   *settingstore.Store
	recognizeStore *recognizestore.Store
	authStore      *authstore.Store

	dictSrv      *dictservice.DictSrv
	personSrv    *personservice.PersonSrv
	bookSrv      *bookservice.BookSrv
	eventSrv     *eventservice.EventSrv
	recordSrv    *recordservice.RecordSrv
	viewSrv      *viewservice.ViewSrv
	statsSrv     *statsservice.StatsSrv
	searchSrv    *searchservice.SearchSrv
	settingSrv   *settingservice.SettingSrv
	recognizeSrv *recognizeservice.RecognizeSrv
	authSrv      *authservice.AuthSrv
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tx := testDB.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	t.Cleanup(func() {
		if err := tx.Rollback().Error; err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
			t.Logf("rollback: %v", err)
		}
	})

	database := rdb.New(tx, "sqlite")
	h := &harness{
		t:              t,
		tx:             tx,
		dictStore:      dictstore.New(),
		personStore:    personstore.New(),
		bookStore:      bookstore.New(),
		eventStore:     eventstore.New(),
		recordStore:    recordstore.New(),
		viewStore:      viewstore.New(),
		statsStore:     statsstore.New(),
		searchStore:    searchstore.New(),
		settingStore:   settingstore.New(),
		recognizeStore: recognizestore.New(),
		authStore:      authstore.New(),
	}
	h.dictSrv = dictservice.NewDictSrv(database, h.dictStore)
	h.personSrv = personservice.NewPersonSrv(database, h.personStore)
	h.bookSrv = bookservice.NewBookSrv(database, h.bookStore, h.dictStore, h.eventStore)
	h.eventSrv = eventservice.NewEventSrv(database, h.eventStore, h.personStore)
	h.recordSrv = recordservice.NewRecordSrv(database, h.recordStore, h.personStore, h.eventStore, h.bookStore, h.dictStore)
	h.settingSrv = settingservice.NewSettingSrv(database, h.settingStore)
	h.viewSrv = viewservice.NewViewSrv(database, h.viewStore, h.recordStore, h.recordSrv, h.personStore, h.eventStore, h.settingStore)
	h.statsSrv = statsservice.NewStatsSrv(database, h.statsStore, h.viewSrv, h.personStore, h.bookStore, h.dictStore)
	h.searchSrv = searchservice.NewSearchSrv(database, h.searchStore, h.recordSrv)
	h.recognizeSrv = recognizeservice.NewRecognizeSrv(database, h.recognizeStore, h.recordSrv, h.personStore, h.recordStore, nil, nil, config.RecognizeConfig{}, t.TempDir())
	authCfg := config.Default()
	authCfg.Server.Mode = "debug"
	authCfg.Auth.Secret = "test-secret-at-least-32-characters-long"
	authCfg.Auth.AdminUserIDs = []uint64{1}
	authSrv, err := authservice.NewAuthSrv(database, h.authStore, &fakeWechat{}, *authCfg)
	requireOK(t, err)
	h.authSrv = authSrv
	return h
}

func (h *harness) ctx() context.Context { return h.ctxAs(1) }

func (h *harness) ctxAs(ownerID uint64) context.Context {
	return middleware.WithOwner(context.Background(), ownerID)
}

// createPerson 通过 service 建亲友，返回 id。
func (h *harness) createPerson(name string) uint64 {
	h.t.Helper()
	detail, err := h.personSrv.Create(h.ctx(), person.CreateParam{Name: name})
	requireOK(h.t, err)
	return detail.ID
}

// createEvent 通过 service 建事项，date 为空时不设日期。
func (h *harness) createEvent(name, eventType, date, precision string) uint64 {
	h.t.Helper()
	if precision == "" {
		precision = "day"
	}
	detail, err := h.eventSrv.Create(h.ctx(), event.CreateParam{
		Name:          name,
		Type:          eventType,
		EventDate:     date,
		DatePrecision: precision,
	})
	requireOK(h.t, err)
	return detail.ID
}

// createBook 通过 service 建礼簿；defaultEventID 为 0 时不设置默认事项。
func (h *harness) createBook(name, category string, defaultEventID uint64) uint64 {
	h.t.Helper()
	detail, err := h.bookSrv.Create(h.ctx(), book.CreateParam{
		Name:           name,
		Category:       category,
		DefaultEventID: defaultEventID,
	})
	requireOK(h.t, err)
	return detail.ID
}

// createRecord 通过 service 建记录，未指定日期时使用 2011-10-01。
func (h *harness) createRecord(param record.CreateParam) *record.Resp {
	h.t.Helper()
	if param.OccurredOn == "" {
		param.OccurredOn = "2011-10-01"
	}
	resp, err := h.recordSrv.Create(h.ctx(), param)
	requireOK(h.t, err)
	return resp
}

func (h *harness) countRecords() int64 {
	h.t.Helper()
	var count int64
	requireOK(h.t, h.tx.Model(&record.Record{}).Count(&count).Error)
	return count
}

func (h *harness) countRepayLinks() int64 {
	h.t.Helper()
	var count int64
	requireOK(h.t, h.tx.Model(&record.Repay{}).Count(&count).Error)
	return count
}

func (h *harness) countEvents() int64 {
	h.t.Helper()
	var count int64
	requireOK(h.t, h.tx.Model(&event.Event{}).Count(&count).Error)
	return count
}

// requireOK 断言没有错误。
func requireOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireCode 断言错误是 ErrI18 且业务码匹配。
func requireCode(t *testing.T, err error, code int64) *response.ErrI18 {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %d, got nil", code)
	}
	var errI18 *response.ErrI18
	if !errors.As(err, &errI18) {
		t.Fatalf("expected *response.ErrI18, got %T: %v", err, err)
	}
	if errI18.Code != code {
		t.Fatalf("expected code %d, got %d (%v)", code, errI18.Code, errI18)
	}
	return errI18
}

// requireReason 断言 409 冲突的 custom.reason。
func requireReason(t *testing.T, err error, reason string) *response.ErrI18 {
	t.Helper()
	errI18 := requireCode(t, err, 409)
	got, _ := errI18.Custom["reason"].(string)
	if got != reason {
		t.Fatalf("expected reason %q, got %q", reason, got)
	}
	return errI18
}

// confirmPayload 取出软确认 data。
func confirmPayload(t *testing.T, errI18 *response.ErrI18) response.ConfirmPayload {
	t.Helper()
	payload, ok := errI18.Data.(response.ConfirmPayload)
	if !ok {
		t.Fatalf("expected response.ConfirmPayload, got %T", errI18.Data)
	}
	return payload
}

// hasReferenceCount 取出 HasReference 的 refCount。
func hasReferenceCount(t *testing.T, errI18 *response.ErrI18) int64 {
	t.Helper()
	data, ok := errI18.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", errI18.Data)
	}
	count, ok := data["refCount"].(int64)
	if !ok {
		t.Fatalf("expected int64 refCount, got %T", data["refCount"])
	}
	return count
}
