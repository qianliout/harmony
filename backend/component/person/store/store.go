package store

import (
	"context"
	"fmt"
	"time"

	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/rdb"
)

type PersonDao struct {
	db *rdb.RDB
}

func NewPersonDao(db *rdb.RDB) *PersonDao {
	return &PersonDao{db: db}
}

func (dal *PersonDao) CreatePerson(ctx context.Context, data *person.Person) error {
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

func (dal *PersonDao) SearchPerson(ctx context.Context, param *person.SearchPersonParam) ([]*person.Person, int64, error) {
	result := make([]*person.Person, 0)
	param = param.Serialize()
	if err := param.Check(); err != nil {
		return result, 0, err
	}

	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()

	tb := (&person.Person{}).TableName()
	db := dal.db.Get().WithContext(cancelCtx).Table(tb)
	db = db.Where("owner_id = ?", param.OwnerID)
	if len(param.PersonIDs) > 0 {
		db = db.Where("id IN ?", param.PersonIDs)
	}
	if param.Name != "" {
		db = db.Where("name = ?", param.Name)
	}
	if len(param.Names) > 0 {
		db = db.Where("name IN ?", param.Names)
	}
	// Keyword 同时匹配姓名、备注的子串与拼音前缀。
	if param.Keyword != "" {
		like := "%" + param.Keyword + "%"
		prefix := param.Keyword + "%"
		db = db.Where("(name LIKE ? OR remark LIKE ? OR name_pinyin LIKE ? OR name_initials LIKE ?)",
			like, like, prefix, prefix)
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

func (dal *PersonDao) UpdatePerson(ctx context.Context, id int64, updater map[string]any) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	if len(updater) == 0 {
		return fmt.Errorf("updater is empty")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&person.Person{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Updates(updater).Error; err != nil {
		return err
	}
	return nil
}

func (dal *PersonDao) DeletePerson(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&person.Person{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Delete(&person.Person{}).Error; err != nil {
		return err
	}
	return nil
}
