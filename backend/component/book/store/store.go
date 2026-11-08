package store

import (
	"gorm.io/gorm"

	"time"

	"harmony/infra/model/book"
	"harmony/infra/model/record"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Get(db *gorm.DB, ownerID uint64, id uint64) (*book.Book, error) {
	var b book.Book
	if err := db.Where("owner_id = ? AND id = ?", ownerID, id).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]book.Book, error) {
	items := make([]book.Book, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := db.Where("owner_id = ? AND id IN ?", ownerID, ids).Find(&items).Error
	return items, err
}

func (s *Store) FindByName(db *gorm.DB, ownerID uint64, name string) (*book.Book, error) {
	var item book.Book
	if err := db.Where("owner_id = ? AND name = ?", ownerID, name).Order("id ASC").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) List(db *gorm.DB, ownerID uint64, status string) ([]book.Book, error) {
	items := make([]book.Book, 0)
	query := db.Where("owner_id = ?", ownerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("status ASC, created_at DESC, id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) Create(db *gorm.DB, b *book.Book) error {
	if b.Status == "" {
		b.Status = "normal"
	}
	return db.Create(b).Error
}

func (s *Store) Update(db *gorm.DB, b *book.Book) error {
	return db.Model(b).Select("name", "category", "default_event_id", "description").Updates(b).Error
}

func (s *Store) Delete(db *gorm.DB, ownerID uint64, id uint64) error {
	return db.Where("owner_id = ? AND id = ?", ownerID, id).Delete(&book.Book{}).Error
}

func (s *Store) CountRecords(db *gorm.DB, ownerID uint64, bookID uint64) (int64, error) {
	var count int64
	err := db.Model(&record.Record{}).Where("owner_id = ? AND book_id = ?", ownerID, bookID).Count(&count).Error
	return count, err
}

type aggRow struct {
	BookID       uint64 `gorm:"column:book_id"`
	RecordCount  int64  `gorm:"column:record_count"`
	InCent       int64  `gorm:"column:in_cent"`
	OutCent      int64  `gorm:"column:out_cent"`
	PendingCount int64  `gorm:"column:pending_count"`
}

func (s *Store) Aggregates(db *gorm.DB, ownerID uint64, bookIDs []uint64) (map[uint64]book.Agg, error) {
	out := make(map[uint64]book.Agg, len(bookIDs))
	if len(bookIDs) == 0 {
		return out, nil
	}
	rows := make([]aggRow, 0)
	err := db.Model(&record.Record{}).
		Select(`book_id,
			COUNT(*) AS record_count,
			COALESCE(SUM(CASE WHEN direction = 'in' THEN total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN direction = 'out' THEN total_cent ELSE 0 END), 0) AS out_cent,
			COALESCE(SUM(CASE WHEN direction = 'in' AND status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_count`).
		Where("owner_id = ? AND book_id IN ?", ownerID, bookIDs).
		Group("book_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.BookID] = book.Agg{
			RecordCount:  row.RecordCount,
			InCent:       row.InCent,
			OutCent:      row.OutCent,
			PendingCount: row.PendingCount,
		}
	}
	return out, nil
}

func (s *Store) ByItem(db *gorm.DB, ownerID uint64, bookID uint64) ([]book.ItemPart, error) {
	parts := make([]book.ItemPart, 0)
	err := db.Model(&record.Record{}).
		Select(`item_kind, direction,
			COALESCE(SUM(total_cent), 0) AS cent,
			COUNT(*) AS record_count`).
		Where("owner_id = ? AND book_id = ?", ownerID, bookID).
		Group("item_kind, direction").
		Order("item_kind ASC, direction ASC").
		Scan(&parts).Error
	return parts, err
}

// GetBooksInTx 在写事务内读取涉及的本子。SQLite 没有行锁，并发安全由 _txlock=immediate 保证。
func (s *Store) GetBooksInTx(tx *gorm.DB, ownerID uint64, ids []uint64) ([]book.Book, error) {
	items := make([]book.Book, 0)
	err := tx.Where("owner_id = ? AND id IN ?", ownerID, ids).Find(&items).Error
	return items, err
}

func (s *Store) MoveRecords(tx *gorm.DB, ownerID uint64, sourceIDs []uint64, targetID uint64) (int64, error) {
	result := tx.Model(&record.Record{}).
		Where("owner_id = ? AND book_id IN ?", ownerID, sourceIDs).
		Update("book_id", targetID)
	return result.RowsAffected, result.Error
}

// MovePageChecks 把按页核对记录迁移到目标本子；目标已有同页时以来源为准。
func (s *Store) MovePageChecks(tx *gorm.DB, ownerID uint64, sourceIDs []uint64, targetID uint64) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	now := time.Now()
	err := tx.Exec(`INSERT INTO paper_page_check (owner_id, book_id, paper_page, declared_cent, created_at, updated_at)
		SELECT owner_id, ?, paper_page, declared_cent, ?, ?
		FROM paper_page_check WHERE owner_id = ? AND book_id IN ? AND true
		ON CONFLICT(owner_id, book_id, paper_page) DO UPDATE SET declared_cent = excluded.declared_cent, updated_at = excluded.updated_at`,
		targetID, now, now, ownerID, sourceIDs).Error
	if err != nil {
		return err
	}
	return tx.Exec("DELETE FROM paper_page_check WHERE owner_id = ? AND book_id IN ?", ownerID, sourceIDs).Error
}

func (s *Store) RepointMergedInto(tx *gorm.DB, ownerID uint64, sourceIDs []uint64, targetID uint64) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	return tx.Model(&book.Book{}).
		Where("owner_id = ? AND merged_into_id IN ?", ownerID, sourceIDs).
		Update("merged_into_id", targetID).Error
}

func (s *Store) MarkMerged(tx *gorm.DB, ownerID uint64, sourceIDs []uint64, targetID uint64) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	return tx.Model(&book.Book{}).
		Where("owner_id = ? AND id IN ?", ownerID, sourceIDs).
		Updates(map[string]any{
			"status":         "merged",
			"merged_into_id": targetID,
			"merged_at":      time.Now(),
		}).Error
}

func (s *Store) CreateMergeLog(tx *gorm.DB, row *book.MergeLogRow) error {
	return tx.Create(row).Error
}

func (s *Store) ListMergeLogs(db *gorm.DB, ownerID uint64) ([]book.MergeLogRow, error) {
	items := make([]book.MergeLogRow, 0)
	err := db.Where("owner_id = ?", ownerID).Order("created_at DESC, id DESC").Find(&items).Error
	return items, err
}

// CountRecordsByBooks 合并前统计将迁移的记录数。
func (s *Store) CountRecordsByBooks(db *gorm.DB, ownerID uint64, bookIDs []uint64) (int64, error) {
	var count int64
	if len(bookIDs) == 0 {
		return 0, nil
	}
	err := db.Model(&record.Record{}).Where("owner_id = ? AND book_id IN ?", ownerID, bookIDs).Count(&count).Error
	return count, err
}

func (s *Store) ItemCountsByBooks(db *gorm.DB, ownerID uint64, bookIDs []uint64) ([]book.ItemCount, error) {
	items := make([]book.ItemCount, 0)
	if len(bookIDs) == 0 {
		return items, nil
	}
	err := db.Model(&record.Record{}).
		Select("item_kind, COUNT(*) AS record_count").
		Where("owner_id = ? AND book_id IN ?", ownerID, bookIDs).
		Group("item_kind").
		Order("item_kind ASC").
		Scan(&items).Error
	return items, err
}
