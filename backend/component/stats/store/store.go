package store

import (

	"gorm.io/gorm"
)

type Totals struct {
	InCent      int64 `gorm:"column:in_cent"`
	OutCent     int64 `gorm:"column:out_cent"`
	RecordCount int64 `gorm:"column:record_count"`
	PersonCount int64 `gorm:"column:person_count"`
}

type MonthRow struct {
	Month       int   `gorm:"column:m"`
	Direction   string `gorm:"column:direction"`
	TotalCent   int64 `gorm:"column:total_cent"`
	RecordCount int64 `gorm:"column:cnt"`
}

type KindRow struct {
	ItemKind    string `gorm:"column:item_kind"`
	Direction   string `gorm:"column:direction"`
	TotalCent   int64  `gorm:"column:total_cent"`
	RecordCount int64  `gorm:"column:cnt"`
}

type EventTypeRow struct {
	Type        string `gorm:"column:type"`
	TotalCent   int64  `gorm:"column:total_cent"`
	RecordCount int64  `gorm:"column:record_count"`
	EventCount  int64  `gorm:"column:event_count"`
}

type PersonRow struct {
	PersonID    uint64 `gorm:"column:person_id"`
	Name        string `gorm:"column:name"`
	Relation    string `gorm:"column:relation"`
	InCent      int64  `gorm:"column:in_cent"`
	OutCent     int64  `gorm:"column:out_cent"`
	RecordCount int64  `gorm:"column:record_count"`
}

type BookRow struct {
	BookID      uint64 `gorm:"column:book_id"`
	Name        string `gorm:"column:name"`
	Category    string `gorm:"column:category"`
	InCent      int64  `gorm:"column:in_cent"`
	OutCent     int64  `gorm:"column:out_cent"`
	RecordCount int64  `gorm:"column:record_count"`
}

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Years(db *gorm.DB, ownerID uint64) ([]int64, error) {
	years := make([]int64, 0)
	err := db.Table("record").
		Select("DISTINCT CAST(substr(occurred_on, 1, 4) AS INTEGER) AS year").
		Where("owner_id = ?", ownerID).
		Order("year DESC").
		Pluck("year", &years).Error
	return years, err
}

func (s *Store) Totals(db *gorm.DB, ownerID uint64, start, end string) (Totals, error) {
	var row Totals
	err := db.Table("record").
		Select(`COALESCE(SUM(CASE WHEN direction = 'in' THEN total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN direction = 'out' THEN total_cent ELSE 0 END), 0) AS out_cent,
			COUNT(*) AS record_count,
			COUNT(DISTINCT person_id) AS person_count`).
		Where("owner_id = ? AND occurred_on >= ? AND occurred_on < ?", ownerID, start, end).
		Scan(&row).Error
	return row, err
}

func (s *Store) Monthly(db *gorm.DB, ownerID uint64, start, end string) ([]MonthRow, error) {
	rows := make([]MonthRow, 0)
	err := db.Table("record").
		Select("CAST(substr(occurred_on, 6, 2) AS INTEGER) AS m, direction, COALESCE(SUM(total_cent), 0) AS total_cent, COUNT(*) AS cnt").
		Where("owner_id = ? AND occurred_on >= ? AND occurred_on < ?", ownerID, start, end).
		Group("m, direction").
		Scan(&rows).Error
	return rows, err
}

func (s *Store) ByItem(db *gorm.DB, ownerID uint64, start, end string) ([]KindRow, error) {
	rows := make([]KindRow, 0)
	err := db.Table("record").
		Select("item_kind, direction, COALESCE(SUM(total_cent), 0) AS total_cent, COUNT(*) AS cnt").
		Where("owner_id = ? AND occurred_on >= ? AND occurred_on < ?", ownerID, start, end).
		Group("item_kind, direction").
		Scan(&rows).Error
	return rows, err
}

type CategoryRow struct {
	Category    string `gorm:"column:category"`
	InCent      int64  `gorm:"column:in_cent"`
	OutCent     int64  `gorm:"column:out_cent"`
	TotalCent   int64  `gorm:"column:total_cent"`
	RecordCount int64  `gorm:"column:record_count"`
}

func (s *Store) ByBookCategory(db *gorm.DB, ownerID uint64, start, end string) ([]CategoryRow, error) {
	rows := make([]CategoryRow, 0)
	err := db.Table("record AS r").
		Select(`b.category AS category,
			COALESCE(SUM(CASE WHEN r.direction = 'in' THEN r.total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN r.direction = 'out' THEN r.total_cent ELSE 0 END), 0) AS out_cent,
			COALESCE(SUM(r.total_cent), 0) AS total_cent,
			COUNT(*) AS record_count`).
		Joins("JOIN book AS b ON b.id = r.origin_book_id AND b.owner_id = r.owner_id").
		Where("r.owner_id = ? AND r.occurred_on >= ? AND r.occurred_on < ?", ownerID, start, end).
		Group("b.category").
		Order("total_cent DESC").
		Scan(&rows).Error
	return rows, err
}

func (s *Store) ByEventType(db *gorm.DB, ownerID uint64, start, end string) ([]EventTypeRow, error) {
	rows := make([]EventTypeRow, 0)
	err := db.Table("record AS r").
		Select(`e.type AS type,
			COALESCE(SUM(r.total_cent), 0) AS total_cent,
			COUNT(*) AS record_count,
			COUNT(DISTINCT r.event_id) AS event_count`).
		Joins("JOIN event AS e ON e.id = r.event_id AND e.owner_id = r.owner_id").
		Where("r.owner_id = ? AND r.occurred_on >= ? AND r.occurred_on < ?", ownerID, start, end).
		Group("e.type").
		Scan(&rows).Error
	return rows, err
}

func (s *Store) TopPersons(db *gorm.DB, ownerID uint64, start, end string, limit int) ([]PersonRow, error) {
	rows := make([]PersonRow, 0)
	err := db.Table("record AS r").
		Select(`r.person_id, p.name, p.relation,
			COALESCE(SUM(CASE WHEN r.direction = 'in' THEN r.total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN r.direction = 'out' THEN r.total_cent ELSE 0 END), 0) AS out_cent,
			COUNT(*) AS record_count`).
		Joins("JOIN person AS p ON p.id = r.person_id AND p.owner_id = r.owner_id").
		Where("r.owner_id = ? AND r.occurred_on >= ? AND r.occurred_on < ?", ownerID, start, end).
		Group("r.person_id, p.name, p.relation").
		Order("(in_cent + out_cent) DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (s *Store) ByBook(db *gorm.DB, ownerID uint64, start, end string) ([]BookRow, error) {
	rows := make([]BookRow, 0)
	err := db.Table("record AS r").
		Select(`r.book_id, b.name, b.category,
			COALESCE(SUM(CASE WHEN r.direction = 'in' THEN r.total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN r.direction = 'out' THEN r.total_cent ELSE 0 END), 0) AS out_cent,
			COUNT(*) AS record_count`).
		Joins("JOIN book AS b ON b.id = r.book_id AND b.owner_id = r.owner_id").
		Where("r.owner_id = ? AND r.occurred_on >= ? AND r.occurred_on < ?", ownerID, start, end).
		Group("r.book_id, b.name, b.category").
		Order("record_count DESC").
		Scan(&rows).Error
	return rows, err
}

func (s *Store) KindByEvent(db *gorm.DB, ownerID uint64, eventID uint64) ([]KindRow, error) {
	rows := make([]KindRow, 0)
	err := db.Table("record").
		Select("item_kind, direction, COALESCE(SUM(total_cent), 0) AS total_cent, COUNT(*) AS cnt").
		Where("owner_id = ? AND event_id = ?", ownerID, eventID).
		Group("item_kind, direction").
		Scan(&rows).Error
	return rows, err
}
