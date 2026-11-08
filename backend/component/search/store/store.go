package store

import (
	"fmt"

	"gorm.io/gorm"

	"harmony/infra/model/search"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Persons(db *gorm.DB, ownerID uint64, keyword string, limit int) ([]search.PersonHit, int64, error) {
	like := "%" + keyword + "%"
	prefix := keyword + "%"
	query := db.Table("person AS p").
		Where("p.owner_id = ? AND (p.name LIKE ? OR p.name_pinyin LIKE ? OR p.name_initials LIKE ?)", ownerID, like, prefix, prefix)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]search.PersonHit, 0)
	err := db.Table("person AS p").
		Select(`p.id, p.name, p.relation,
			(SELECT COUNT(*) FROM record r WHERE r.owner_id = p.owner_id AND r.person_id = p.id) AS record_count`).
		Where("p.owner_id = ? AND (p.name LIKE ? OR p.name_pinyin LIKE ? OR p.name_initials LIKE ?)", ownerID, like, prefix, prefix).
		Order("record_count DESC, p.id ASC").
		Limit(limit).
		Scan(&items).Error
	return items, total, err
}

func (s *Store) Events(db *gorm.DB, ownerID uint64, keyword string, limit int) ([]search.EventHit, int64, error) {
	query := db.Table("event").
		Where("owner_id = ? AND placeholder = 0 AND name LIKE ?", ownerID, "%"+keyword+"%")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]search.EventHit, 0)
	err := query.Select("id, name, type, event_date, host").
		Order("event_date IS NULL ASC, event_date DESC, id DESC").
		Limit(limit).
		Scan(&items).Error
	return items, total, err
}

func (s *Store) EventsByYear(db *gorm.DB, ownerID uint64, year int64, limit int) ([]search.EventHit, int64, error) {
	start := fmt.Sprintf("%04d-01-01", year)
	end := fmt.Sprintf("%04d-01-01", year+1)
	query := db.Table("event").
		Where("owner_id = ? AND placeholder = 0 AND event_date >= ? AND event_date < ?", ownerID, start, end)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]search.EventHit, 0)
	err := query.Select("id, name, type, event_date, host").
		Order("event_date DESC, id DESC").
		Limit(limit).
		Scan(&items).Error
	return items, total, err
}

func (s *Store) Books(db *gorm.DB, ownerID uint64, keyword string, limit int) ([]search.BookHit, int64, error) {
	query := db.Table("book").
		Joins("LEFT JOIN dict_item AS d ON d.owner_id = book.owner_id AND d.dict_type = 'book_category' AND d.code = book.category").
		Where("book.owner_id = ? AND book.status = ? AND book.name LIKE ?", ownerID, "normal", "%"+keyword+"%")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]search.BookHit, 0)
	err := query.Select(`book.id, book.name, book.category, COALESCE(d.label, book.category) AS category_label,
			(SELECT COUNT(*) FROM record r WHERE r.owner_id = book.owner_id AND r.book_id = book.id) AS record_count`).
		Order("record_count DESC, book.id DESC").
		Limit(limit).
		Scan(&items).Error
	return items, total, err
}

// FillPersonLabels 补齐关系文案，避免 service 再查一次字典。
func FillPersonLabels(items []search.PersonHit) {
	for i := range items {
		items[i].RelationLabel = relationLabel(items[i].Relation)
	}
}

func relationLabel(relation string) string {
	switch relation {
	case "relative":
		return "亲戚"
	case "friend":
		return "朋友"
	case "colleague":
		return "同事"
	case "neighbor":
		return "邻居"
	default:
		return "其他"
	}
}
