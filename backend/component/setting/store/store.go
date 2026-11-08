package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harmony/infra/model/setting"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Get(db *gorm.DB, ownerID uint64) (map[string]string, error) {
	rows := make([]setting.Row, 0)
	if err := db.Where("owner_id = ?", ownerID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for i := range rows {
		out[rows[i].K] = rows[i].V
	}
	return out, nil
}

func (s *Store) Upsert(db *gorm.DB, ownerID uint64, key, value string) error {
	row := &setting.Row{
		OwnerID:   ownerID,
		K:         key,
		V:         value,
		UpdatedAt: time.Now(),
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_id"}, {Name: "k"}},
		DoUpdates: clause.AssignmentColumns([]string{"v", "updated_at"}),
	}).Create(row).Error
}
