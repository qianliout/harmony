package store

import (
	"context"
	"fmt"
	"time"

	"harmony/infra/model/book"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/rdb"
)

type BookDao struct {
	db *rdb.RDB
}

func NewBookDao(db *rdb.RDB) *BookDao {
	return &BookDao{db: db}
}

func (dal *BookDao) CreateBook(ctx context.Context, data *book.Book) error {
	data = data.Serialize()
	if err := data.Check(); err != nil {
		return err
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*10)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table(data.TableName())
	if err := db.Create(data).Error; err != nil {
		return err
	}
	return nil
}

func (dal *BookDao) SearchBook(ctx context.Context, param *book.SearchBookParam) ([]*book.Book, int64, error) {
	result := make([]*book.Book, 0)
	param = param.Serialize()
	if err := param.Check(); err != nil {
		return result, 0, err
	}

	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()

	tb := (&book.Book{}).TableName()
	db := dal.db.Get().WithContext(cancelCtx).Table(tb)
	db = db.Where("owner_id = ?", param.OwnerID)
	if len(param.BookIDs) > 0 {
		db = db.Where("id IN ?", param.BookIDs)
	}
	if param.Category != "" {
		db = db.Where("category = ?", param.Category)
	}
	if param.Keyword != "" {
		like := "%" + param.Keyword + "%"
		db = db.Where("(name LIKE ? OR description LIKE ?)", like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return result, 0, err
	}
	// 增加 排序 / limit / offset
	db = modelfilter.AddFilter(db, param.Filter)

	if err := db.Find(&result).Error; err != nil {
		return result, 0, err
	}
	for _, row := range result {
		row.Deserialize()
	}
	return result, total, nil
}

func (dal *BookDao) UpdateBook(ctx context.Context, id int64, updater map[string]any) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	if len(updater) == 0 {
		return fmt.Errorf("updater is empty")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&book.Book{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Updates(updater).Error; err != nil {
		return err
	}
	return nil
}

func (dal *BookDao) DeleteBook(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&book.Book{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Delete(&book.Book{}).Error; err != nil {
		return err
	}
	return nil
}
