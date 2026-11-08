package store

import (
	"time"

	"gorm.io/gorm"

	"harmony/infra/model/recognize"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Create(db *gorm.DB, task *recognize.Task) error {
	return db.Create(task).Error
}

func (s *Store) Get(db *gorm.DB, ownerID uint64, id uint64) (*recognize.Task, error) {
	var task recognize.Task
	if err := db.Where("owner_id = ? AND id = ?", ownerID, id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Store) List(db *gorm.DB, ownerID uint64, offset, limit int64) ([]recognize.Task, int64, error) {
	query := db.Model(&recognize.Task{}).Where("owner_id = ?", ownerID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]recognize.Task, 0)
	if err := query.Order("created_at DESC, id DESC").Offset(int(offset)).Limit(int(limit)).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Store) MarkRunning(db *gorm.DB, id uint64) error {
	return db.Model(&recognize.Task{}).Where("id = ?", id).
		Updates(map[string]any{"status": "running", "error_message": ""}).Error
}

func (s *Store) MarkSucceeded(db *gorm.DB, id uint64, itemsJSON string, raw string) error {
	now := time.Now()
	return db.Model(&recognize.Task{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":        "succeeded",
			"items":         itemsJSON,
			"raw_response":  raw,
			"error_message": "",
			"finished_at":   now,
		}).Error
}

func (s *Store) MarkFailed(db *gorm.DB, id uint64, message string, raw string) error {
	now := time.Now()
	return db.Model(&recognize.Task{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":        "failed",
			"error_message": message,
			"raw_response":  raw,
			"finished_at":   now,
		}).Error
}

func (s *Store) MarkCommitted(db *gorm.DB, id uint64) error {
	return db.Model(&recognize.Task{}).Where("id = ?", id).Update("committed", true).Error
}

// FailStale 在服务重启时把未完成的任务置为失败，提示用户重试。
func (s *Store) FailStale(db *gorm.DB) error {
	return db.Model(&recognize.Task{}).
		Where("status IN ?", []string{"pending", "running"}).
		Updates(map[string]any{
			"status":        "failed",
			"error_message": "服务重启，识别未完成，请重新上传",
			"finished_at":   time.Now(),
		}).Error
}
