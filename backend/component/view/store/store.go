package store

import (

	"gorm.io/gorm"
)

type PendingRow struct {
	PersonID  uint64    `gorm:"column:person_id"`
	Count     int64     `gorm:"column:cnt"`
	TotalCent int64     `gorm:"column:total_cent"`
	Earliest  string `gorm:"column:earliest"`
}

type Store struct{}

func New() *Store { return &Store{} }

// Owed 是我欠别人：收方向且待还，按人聚合。
func (s *Store) Owed(db *gorm.DB, ownerID uint64) ([]PendingRow, error) {
	rows := make([]PendingRow, 0)
	err := db.Table("record").
		Select("person_id, COUNT(*) AS cnt, COALESCE(SUM(total_cent), 0) AS total_cent, MIN(occurred_on) AS earliest").
		Where("owner_id = ? AND direction = ? AND status = ?", ownerID, "in", "pending").
		Group("person_id").
		Order("earliest ASC").
		Scan(&rows).Error
	return rows, err
}

// Owing 是别人欠我：送方向待还，且没有被关联为还礼。
func (s *Store) Owing(db *gorm.DB, ownerID uint64) ([]PendingRow, error) {
	rows := make([]PendingRow, 0)
	err := db.Table("record AS r").
		Select("r.person_id, COUNT(*) AS cnt, COALESCE(SUM(r.total_cent), 0) AS total_cent, MIN(r.occurred_on) AS earliest").
		Where("r.owner_id = ? AND r.direction = ? AND r.status = ?", ownerID, "out", "pending").
		Where("NOT EXISTS (SELECT 1 FROM record_repay rp WHERE rp.repay_record_id = r.id)").
		Group("r.person_id").
		Order("earliest ASC").
		Scan(&rows).Error
	return rows, err
}
