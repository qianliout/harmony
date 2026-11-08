package test

import (
	"testing"

	"harmony/infra/model/book"
	"harmony/infra/model/record"
)

// TestPaperPageCheck 对应 docs/02「按页核对」：
// 录入 15 笔共 1070 元、纸本合计 1070 → consistent；删除一笔后 inconsistent。
func TestPaperPageCheck(t *testing.T) {
	h := newHarness(t)
	personID := h.createPerson("张三")
	eventID := h.createEvent("升学宴", "school", "2011-08-20", "day")
	bookID := h.createBook("人情礼本", "renqing", 0)

	// 14 笔 70 元 + 1 笔 90 元 = 1070 元 = 107000 分。
	amounts := make([]int64, 0, 15)
	for range 14 {
		amounts = append(amounts, 7000)
	}
	amounts = append(amounts, 9000)

	var lastID uint64
	for i, amount := range amounts {
		created := h.createRecord(record.CreateParam{
			PersonID: personID, EventID: eventID, BookID: bookID,
			AmountCent: amount, Direction: "in", OccurredOn: "2011-08-20",
			PaperPage: "1-10", PaperSeq: i + 1, Confirm: true,
		})
		lastID = created.ID
	}

	requireOK(t, h.recordSrv.SetPageCheck(h.ctx(), bookID, "1-10", 107000))
	pages, err := h.recordSrv.Pages(h.ctx(), bookID)
	requireOK(t, err)
	page := findPage(t, pages, "1-10")
	if !page.HasDeclared {
		t.Fatalf("expected declared page 1-10")
	}
	if page.RecordCent != 107000 || page.RecordCount != 15 {
		t.Fatalf("expected 15 records / 107000 cent, got %d / %d", page.RecordCount, page.RecordCent)
	}
	if !page.Consistent || page.DiffCent != 0 {
		t.Fatalf("expected consistent, got consistent=%v diff=%d", page.Consistent, page.DiffCent)
	}

	requireOK(t, h.recordSrv.Delete(h.ctx(), lastID, false))
	pages, err = h.recordSrv.Pages(h.ctx(), bookID)
	requireOK(t, err)
	page = findPage(t, pages, "1-10")
	if page.Consistent {
		t.Fatalf("expected inconsistent after deleting a record")
	}
	if page.DiffCent != -9000 {
		t.Fatalf("expected diff -9000, got %d", page.DiffCent)
	}
	if page.RecordCount != 14 || page.RecordCent != 98000 {
		t.Fatalf("expected 14 records / 98000 cent after delete, got %d / %d", page.RecordCount, page.RecordCent)
	}
}

func findPage(t *testing.T, pages []book.PaperPage, page string) book.PaperPage {
	t.Helper()
	for _, item := range pages {
		if item.Page == page {
			return item
		}
	}
	t.Fatalf("page %q not found in %+v", page, pages)
	return book.PaperPage{}
}
