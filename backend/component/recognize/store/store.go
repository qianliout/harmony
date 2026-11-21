package store

import (
	"context"
	"fmt"
	"time"

	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/recognize"
	"harmony/infra/rdb"
)

type RecognizeTaskDao struct {
	db *rdb.RDB
}

func NewRecognizeTaskDao(db *rdb.RDB) *RecognizeTaskDao {
	return &RecognizeTaskDao{db: db}
}

func (dal *RecognizeTaskDao) CreateRecognizeTask(ctx context.Context, data *recognize.RecognizeTask) error {
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

func (dal *RecognizeTaskDao) SearchRecognizeTask(ctx context.Context, param *recognize.SearchRecognizeTaskParam) ([]*recognize.RecognizeTask, int64, error) {
	result := make([]*recognize.RecognizeTask, 0)
	param = param.Serialize()
	if err := param.Check(); err != nil {
		return result, 0, err
	}

	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()

	tb := (&recognize.RecognizeTask{}).TableName()
	db := dal.db.Get().WithContext(cancelCtx).Table(tb)
	// OwnerID 为 0 表示不限 owner，仅用于启动时清理残留任务。
	if param.OwnerID > 0 {
		db = db.Where("owner_id = ?", param.OwnerID)
	}
	if len(param.TaskIDs) > 0 {
		db = db.Where("id IN ?", param.TaskIDs)
	}
	if len(param.Statuses) > 0 {
		db = db.Where("status IN ?", param.Statuses)
	}
	if param.CreatedFrom > 0 {
		db = db.Where("created_at >= ?", param.CreatedFrom)
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

func (dal *RecognizeTaskDao) UpdateRecognizeTask(ctx context.Context, id int64, updater map[string]any) error {
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	if len(updater) == 0 {
		return fmt.Errorf("updater is empty")
	}
	cancelCtx, cancelFunc := context.WithTimeout(ctx, time.Second*3)
	defer cancelFunc()
	db := dal.db.Get().WithContext(cancelCtx).Table((&recognize.RecognizeTask{}).TableName())
	db = db.Where("id = ?", id)
	if err := db.Updates(updater).Error; err != nil {
		return err
	}
	return nil
}
