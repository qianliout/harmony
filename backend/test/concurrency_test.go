package test

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	bookservice "harmony/component/book/service"
	bookstore "harmony/component/book/store"
	dictservice "harmony/component/dict/service"
	dictstore "harmony/component/dict/store"
	eventservice "harmony/component/event/service"
	eventstore "harmony/component/event/store"
	personservice "harmony/component/person/service"
	personstore "harmony/component/person/store"
	recordservice "harmony/component/record/service"
	recordstore "harmony/component/record/store"
	"harmony/infra/middleware"
	"harmony/infra/model/book"
	"harmony/infra/model/event"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/migrate"
)

// TestConcurrentRepaySQLite 验证 10 个并发回礼只有一个成功，其余返回 409，且没有 database is locked。
func TestConcurrentRepaySQLite(t *testing.T) {
	database, err := rdb.Open(filepath.Join(t.TempDir(), "concurrent.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.Transaction(func(tx *gorm.DB) error {
		for _, statement := range migrate.Statements() {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("schema: %v", err)
	}

	dictStore := dictstore.New()
	personStore := personstore.New()
	bookStore := bookstore.New()
	eventStore := eventstore.New()
	recordStore := recordstore.New()
	dictSrv := dictservice.NewDictSrv(database, dictStore)
	personSrv := personservice.NewPersonSrv(database, personStore)
	bookSrv := bookservice.NewBookSrv(database, bookStore, dictStore, eventStore)
	eventSrv := eventservice.NewEventSrv(database, eventStore, personStore)
	recordSrv := recordservice.NewRecordSrv(database, recordStore, personStore, eventStore, bookStore, dictStore)
	_ = dictSrv

	ctx := middleware.WithOwner(t.Context(), 1)
	p, err := personSrv.Create(ctx, person.CreateParam{Name: "并发人"})
	requireOK(t, err)
	ev, err := eventSrv.Create(ctx, event.CreateParam{Name: "并发事", Type: "wedding", EventDate: "2024-01-01", DatePrecision: "day"})
	requireOK(t, err)
	bk, err := bookSrv.Create(ctx, book.CreateParam{Name: "并发本", Category: "renqing"})
	requireOK(t, err)
	origin, err := recordSrv.Create(ctx, record.CreateParam{
		PersonID: p.ID, EventID: ev.ID, BookID: bk.ID, Direction: "in",
		AmountCent: 10000, OccurredOn: "2024-01-01", DatePrecision: "day", Status: "pending",
	})
	requireOK(t, err)

	var success, conflict int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := recordSrv.Repay(ctx, record.RepayParam{
				OriginIDs:  []uint64{origin.ID},
				OccurredOn: "2024-02-02",
				Items:      []record.RepayItem{{BookID: bk.ID, ItemKind: "cash", AmountCent: 10000}},
			})
			if err == nil {
				atomic.AddInt32(&success, 1)
				return
			}
			var errI18 *response.ErrI18
			if errors.As(err, &errI18) && errI18.Code == 409 {
				atomic.AddInt32(&conflict, 1)
				return
			}
			t.Errorf("unexpected concurrent repay error: %v", err)
		}()
	}
	wg.Wait()
	if success != 1 || conflict != 9 {
		t.Fatalf("expected 1 success and 9 conflicts, got success=%d conflict=%d", success, conflict)
	}
}
