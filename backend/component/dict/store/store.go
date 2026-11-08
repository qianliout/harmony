package store

import (
	"gorm.io/gorm"

	"harmony/infra/model/book"
	"harmony/infra/model/dict"
	"harmony/infra/model/record"
)

// Store 只做字典项的持久化，事务句柄由 service 传入。
type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) List(db *gorm.DB, ownerID uint64, dictType string) ([]dict.Item, error) {
	items := make([]dict.Item, 0)
	query := db.Where("owner_id = ?", ownerID)
	if dictType != "" {
		query = query.Where("dict_type = ?", dictType)
	}
	if err := query.Order("dict_type ASC, sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) Get(db *gorm.DB, ownerID uint64, id uint64) (*dict.Item, error) {
	var item dict.Item
	if err := db.Where("owner_id = ? AND id = ?", ownerID, id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) GetByCode(db *gorm.DB, ownerID uint64, dictType, code string) (*dict.Item, error) {
	var item dict.Item
	if err := db.Where("owner_id = ? AND dict_type = ? AND code = ?", ownerID, dictType, code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) Create(db *gorm.DB, item *dict.Item) error {
	return db.Create(item).Error
}

func (s *Store) Update(db *gorm.DB, item *dict.Item) error {
	return db.Model(item).Select("label", "color", "sort").Updates(item).Error
}

func (s *Store) Delete(db *gorm.DB, ownerID uint64, id uint64) error {
	return db.Where("owner_id = ? AND id = ?", ownerID, id).Delete(&dict.Item{}).Error
}

// CountBookCategoryRefs 统计礼簿类别被引用次数。
func (s *Store) CountBookCategoryRefs(db *gorm.DB, ownerID uint64, code string) (int64, error) {
	var count int64
	err := db.Model(&book.Book{}).Where("owner_id = ? AND category = ?", ownerID, code).Count(&count).Error
	return count, err
}

// CountItemKindRefs 统计名目被记录引用次数。
func (s *Store) CountItemKindRefs(db *gorm.DB, ownerID uint64, code string) (int64, error) {
	var count int64
	err := db.Model(&record.Record{}).Where("owner_id = ? AND item_kind = ?", ownerID, code).Count(&count).Error
	return count, err
}
