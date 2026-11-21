package service

import (
	"context"
	"fmt"

	"harmony/component/book/store"
	recordstore "harmony/component/record/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/model/book"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/record"
	"harmony/infra/response"
)

type BookSrv struct {
	bookDal   *store.BookDao
	recordDal *recordstore.RecordDao
	log       *logger.Logger
}

func NewBookSrv(bookDal *store.BookDao, recordDal *recordstore.RecordDao) *BookSrv {
	return &BookSrv{
		bookDal:   bookDal,
		recordDal: recordDal,
		log:       logger.New(logger.WithModule("book"), logger.WithSubModule("service")),
	}
}

func (s *BookSrv) SearchBook(ctx context.Context, param *book.SearchBookParam) ([]*book.BookView, int64, error) {
	result := make([]*book.BookView, 0)
	rows, total, err := s.bookDal.SearchBook(ctx, param)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Msg("searchBook failed")
		return result, 0, err
	}
	stats, err := s.statBooks(ctx, param.OwnerID, rows)
	if err != nil {
		return result, 0, err
	}
	result = s.toBookViews(rows, stats)
	return result, total, nil
}

func (s *BookSrv) GetBook(ctx context.Context, param *book.GetBookParam) (*book.BookView, error) {
	row, err := s.getBook(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	rows := []*book.Book{row}
	stats, err := s.statBooks(ctx, param.OwnerID, rows)
	if err != nil {
		return nil, err
	}
	views := s.toBookViews(rows, stats)
	return views[0], nil
}

func (s *BookSrv) CreateBook(ctx context.Context, param *book.CreateBookParam) (*book.BookView, error) {
	row := &book.Book{
		OwnerID:     param.OwnerID,
		Name:        param.Name,
		Category:    param.Category,
		Description: param.Description,
	}
	if err := s.bookDal.CreateBook(ctx, row); err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Str("name", param.Name).Msg("createBook failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("bookID", row.ID).Msg("book created")
	views := s.toBookViews([]*book.Book{row}, map[int64]*record.RecordStat{})
	return views[0], nil
}

// UpdateBook 是完整更新；种类为空时由 Serialize 补默认值，因此走 Serialize → Check → ToUpdater。
func (s *BookSrv) UpdateBook(ctx context.Context, param *book.UpdateBookParam) (*book.BookView, error) {
	before, err := s.getBook(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	after := &book.Book{
		ID:          before.ID,
		OwnerID:     before.OwnerID,
		Name:        param.Name,
		Category:    param.Category,
		Description: param.Description,
	}
	after = after.Serialize()
	if err := after.Check(); err != nil {
		return nil, response.NewErr(err)
	}
	if err := s.bookDal.UpdateBook(ctx, param.ID, after.ToUpdater()); err != nil {
		s.log.Error(ctx).Err(err).Int64("bookID", param.ID).Msg("updateBook failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("bookID", param.ID).Msg("book updated")
	view, err := s.GetBook(ctx, &book.GetBookParam{ID: param.ID, OwnerID: param.OwnerID})
	if err != nil {
		return nil, err
	}
	return view, nil
}

// DeleteBook 有关联记录时阻止删除。
func (s *BookSrv) DeleteBook(ctx context.Context, param *book.GetBookParam) error {
	if _, err := s.getBook(ctx, param.OwnerID, param.ID); err != nil {
		return err
	}
	refParam := &record.SearchRecordParam{
		OwnerID: param.OwnerID,
		BookID:  param.ID,
		Filter:  &modelfilter.Filter{Limit: 1},
	}
	_, refCount, err := s.recordDal.SearchRecord(ctx, refParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("bookID", param.ID).Msg("count book records failed")
		return err
	}
	if refCount > 0 {
		zh := fmt.Sprintf("这本礼簿下还有 %d 条记录，先处理记录再删除", refCount)
		return response.HasReference(zh, "book has records", refCount)
	}
	if err := s.bookDal.DeleteBook(ctx, param.ID); err != nil {
		s.log.Error(ctx).Err(err).Int64("bookID", param.ID).Msg("deleteBook failed")
		return err
	}
	s.log.Info(ctx).Int64("bookID", param.ID).Msg("book deleted")
	return nil
}

func (s *BookSrv) getBook(ctx context.Context, ownerID, bookID int64) (*book.Book, error) {
	searchParam := &book.SearchBookParam{OwnerID: ownerID, BookIDs: []int64{bookID}}
	rows, _, err := s.bookDal.SearchBook(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("bookID", bookID).Msg("search book by id failed")
		return nil, err
	}
	if len(rows) == 0 {
		return nil, response.NotFound("礼簿不存在", "book not found")
	}
	return rows[0], nil
}

func (s *BookSrv) statBooks(ctx context.Context, ownerID int64, rows []*book.Book) (map[int64]*record.RecordStat, error) {
	stats := make(map[int64]*record.RecordStat, len(rows))
	if len(rows) == 0 {
		return stats, nil
	}
	bookIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		bookIDs = append(bookIDs, row.ID)
	}
	statParam := &record.StatRecordParam{
		OwnerID: ownerID,
		BookIDs: bookIDs,
		GroupBy: consts.RecordGroupByBook,
	}
	statRows, err := s.recordDal.StatRecord(ctx, statParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", ownerID).Msg("stat book records failed")
		return stats, err
	}
	for _, stat := range statRows {
		stats[stat.GroupID] = stat
	}
	return stats, nil
}

func (s *BookSrv) toBookViews(rows []*book.Book, stats map[int64]*record.RecordStat) []*book.BookView {
	views := make([]*book.BookView, 0, len(rows))
	for _, row := range rows {
		view := &book.BookView{
			ID:            row.ID,
			Name:          row.Name,
			Category:      row.Category,
			CategoryLabel: consts.CategoryLabel(row.Category),
			Description:   row.Description,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
		stat, ok := stats[row.ID]
		if ok {
			view.RecordCount = stat.RecordCount
			view.InCent = stat.InCent
		}
		views = append(views, view)
	}
	return views
}
