package store

import (
	"context"
	"fmt"
	"time"

	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/rdb"
)

type RecordDao struct {
	db *rdb.RDB
}

func NewRecordDao(db *rdb.RDB) *RecordDao {
	return &RecordDao{db: db}
}

func (dal *RecordDao) CreateRecord(ctx context.Context, data *record.Record) error {
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

func (dal *RecordDao) SearchRecord(ctx context.Context, param *record.SearchRecordParam) ([]*record.Record, int64, error) {
	result := make([]*record.Record, 0)
	param = param.Serialize()
	if err := param.Check(); err != nil {
		return result, 0, err
	}

	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()

	tb := (&record.Record{}).TableName()
	db := dal.db.Get().WithContext(cancelCtx).Table(tb)
	db = db.Where("owner_id = ?", param.OwnerID)
	if len(param.RecordIDs) > 0 {
		db = db.Where("id IN ?", param.RecordIDs)
	}
	if param.PersonID > 0 {
		db = db.Where("person_id = ?", param.PersonID)
	}
	if param.BookID > 0 {
		db = db.Where("book_id = ?", param.BookID)
	}
	if param.Direction != "" {
		db = db.Where("direction = ?", param.Direction)
	}
	// occurred_on 固定存 YYYY-MM-DD，字典序即时间序。
	if param.From != "" {
		db = db.Where("occurred_on >= ?", param.From)
	}
	if param.To != "" {
		db = db.Where("occurred_on <= ?", param.To)
	}
	// Keyword 匹配备注、事情，以及姓名、备注或拼音命中的亲友。
	if param.Keyword != "" {
		like := "%" + param.Keyword + "%"
		prefix := param.Keyword + "%"
		personTb := (&person.Person{}).TableName()
		db = db.Where("(note LIKE ? OR matter LIKE ? OR person_id IN (SELECT id FROM "+personTb+
			" WHERE owner_id = ? AND (name LIKE ? OR remark LIKE ? OR name_pinyin LIKE ? OR name_initials LIKE ?)))",
			like, like, param.OwnerID, like, like, prefix, prefix)
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

// StatRecord 按与 SearchRecord 相同的筛选口径汇总条数、收送金额与最近日期，可按亲友或礼簿分组。
func (dal *RecordDao) StatRecord(ctx context.Context, param *record.StatRecordParam) ([]*record.RecordStat, error) {
	result := make([]*record.RecordStat, 0)
	param = param.Serialize()
	if err := param.Check(); err != nil {
		return result, err
	}

	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()

	tb := (&record.Record{}).TableName()
	db := dal.db.Get().WithContext(cancelCtx).Table(tb)
	db = db.Where("owner_id = ?", param.OwnerID)
	if param.PersonID > 0 {
		db = db.Where("person_id = ?", param.PersonID)
	}
	if param.BookID > 0 {
		db = db.Where("book_id = ?", param.BookID)
	}
	if len(param.PersonIDs) > 0 {
		db = db.Where("person_id IN ?", param.PersonIDs)
	}
	if len(param.BookIDs) > 0 {
		db = db.Where("book_id IN ?", param.BookIDs)
	}
	if param.Direction != "" {
		db = db.Where("direction = ?", param.Direction)
	}
	if param.From != "" {
		db = db.Where("occurred_on >= ?", param.From)
	}
	if param.To != "" {
		db = db.Where("occurred_on <= ?", param.To)
	}
	if param.Keyword != "" {
		like := "%" + param.Keyword + "%"
		prefix := param.Keyword + "%"
		personTb := (&person.Person{}).TableName()
		db = db.Where("(note LIKE ? OR matter LIKE ? OR person_id IN (SELECT id FROM "+personTb+
			" WHERE owner_id = ? AND (name LIKE ? OR remark LIKE ? OR name_pinyin LIKE ? OR name_initials LIKE ?)))",
			like, like, param.OwnerID, like, like, prefix, prefix)
	}

	// GroupBy 已由 Check 限定为空、person_id 或 book_id，可以直接拼进列名。
	groupColumn := "0"
	if param.GroupBy != consts.RecordGroupByNone {
		groupColumn = param.GroupBy
	}
	db = db.Select(groupColumn + ` AS group_id,
		COUNT(*) AS record_count,
		COALESCE(SUM(CASE WHEN direction = 'in' THEN amount_cent ELSE 0 END), 0) AS in_cent,
		COALESCE(SUM(CASE WHEN direction = 'out' THEN amount_cent ELSE 0 END), 0) AS out_cent,
		COALESCE(MAX(occurred_on), '') AS last_on`)
	if param.GroupBy != consts.RecordGroupByNone {
		db = db.Group(param.GroupBy)
	}

	if err := db.Scan(&result).Error; err != nil {
		return result, err
	}
	return result, nil
}

func (dal *RecordDao) UpdateRecord(ctx context.Context, id int64, updater map[string]any) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	if len(updater) == 0 {
		return fmt.Errorf("updater is empty")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&record.Record{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Updates(updater).Error; err != nil {
		return err
	}
	return nil
}

func (dal *RecordDao) DeleteRecord(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&record.Record{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Delete(&record.Record{}).Error; err != nil {
		return err
	}
	return nil
}
