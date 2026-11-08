package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/record"
	"harmony/infra/utils"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Get(db *gorm.DB, ownerID uint64, id uint64) (*record.Record, error) {
	var item record.Record
	if err := db.Where("owner_id = ? AND id = ?", ownerID, id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]record.Record, error) {
	items := make([]record.Record, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := db.Where("owner_id = ? AND id IN ?", ownerID, ids).Find(&items).Error
	return items, err
}

func (s *Store) List(db *gorm.DB, ownerID uint64, q record.Query, f *modelfilter.Filter) ([]record.Record, int64, error) {
	query := s.filtered(db, ownerID, q)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]record.Record, 0)
	query = query.Order("occurred_on DESC, id DESC")
	if f != nil {
		query = query.Offset(int(f.Offset)).Limit(int(f.Limit))
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Store) filtered(db *gorm.DB, ownerID uint64, q record.Query) *gorm.DB {
	query := db.Model(&record.Record{}).Where("owner_id = ?", ownerID)
	if q.BookID > 0 {
		query = query.Where("book_id = ?", q.BookID)
	}
	if q.OriginBookID > 0 {
		query = query.Where("origin_book_id = ?", q.OriginBookID)
	}
	if q.PersonID > 0 {
		query = query.Where("person_id = ?", q.PersonID)
	}
	if q.EventID > 0 {
		query = query.Where("event_id = ?", q.EventID)
	}
	if q.ItemKind != "" {
		query = query.Where("item_kind = ?", q.ItemKind)
	}
	if q.Direction != "" {
		query = query.Where("direction = ?", q.Direction)
	}
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	if q.From != "" {
		if from, err := utils.ParseDate(q.From); err == nil {
			query = query.Where("occurred_on >= ?", utils.DateString(from))
		}
	}
	if q.To != "" {
		if to, err := utils.ParseDate(q.To); err == nil {
			query = query.Where("occurred_on <= ?", utils.DateString(to))
		}
	}
	intent := q.Intent
	if intent.Kind == "" {
		intent = record.ParseIntent(q.Keyword)
	}
	switch intent.Kind {
	case record.IntentAmount:
		query = query.Where("total_cent = ?", intent.AmountCent)
	case record.IntentAmountRange:
		query = query.Where("total_cent >= ? AND total_cent <= ?", intent.AmountMinCent, intent.AmountMaxCent)
	case record.IntentYear:
		start, end := utils.YearRange(intent.Year)
		query = query.Where("occurred_on >= ? AND occurred_on < ?", utils.DateString(start), utils.DateString(end))
	case record.IntentYearAmount:
		start, end := utils.YearRange(intent.Year)
		query = query.Where("(total_cent = ? OR (occurred_on >= ? AND occurred_on < ?))",
			intent.AmountCent, utils.DateString(start), utils.DateString(end))
	case record.IntentPage:
		query = query.Where("paper_page = ?", intent.PaperPage)
	default:
		if intent.Text != "" {
			like := "%" + intent.Text + "%"
			query = query.Where("(remark LIKE ? OR gift_name LIKE ? OR paper_note LIKE ?)", like, like, like)
		}
	}
	return query
}

func (s *Store) Duplicates(db *gorm.DB, ownerID uint64, personID, eventID, bookID uint64, itemKind string) ([]record.Record, error) {
	items := make([]record.Record, 0)
	err := db.Where("owner_id = ? AND person_id = ? AND event_id = ? AND book_id = ? AND item_kind = ?",
		ownerID, personID, eventID, bookID, itemKind).
		Order("occurred_on DESC, id DESC").
		Find(&items).Error
	return items, err
}

func (s *Store) Create(db *gorm.DB, item *record.Record) error {
	return db.Create(item).Error
}

func (s *Store) Update(db *gorm.DB, item *record.Record) error {
	return db.Save(item).Error
}

func (s *Store) Delete(db *gorm.DB, ownerID uint64, id uint64) error {
	return db.Where("owner_id = ? AND id = ?", ownerID, id).Delete(&record.Record{}).Error
}

func (s *Store) LockRecords(tx *gorm.DB, ownerID uint64, ids []uint64) ([]record.Record, error) {
	items := make([]record.Record, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("owner_id = ? AND id IN ?", ownerID, ids).
		Find(&items).Error
	return items, err
}

func (s *Store) Link(db *gorm.DB, repayRecordID, originRecordID, ownerID uint64) error {
	link := &record.Repay{
		OwnerID:        ownerID,
		RepayRecordID:  repayRecordID,
		OriginRecordID: originRecordID,
	}
	return db.Create(link).Error
}

func (s *Store) LinksByOrigin(db *gorm.DB, ownerID uint64, originID uint64) ([]record.Repay, error) {
	items := make([]record.Repay, 0)
	err := db.Where("owner_id = ? AND origin_record_id = ?", ownerID, originID).Find(&items).Error
	return items, err
}

func (s *Store) LinksByRepay(db *gorm.DB, ownerID uint64, repayID uint64) ([]record.Repay, error) {
	items := make([]record.Repay, 0)
	err := db.Where("owner_id = ? AND repay_record_id = ?", ownerID, repayID).Find(&items).Error
	return items, err
}

func (s *Store) CountLinksByOrigin(db *gorm.DB, ownerID uint64, originID uint64) (int64, error) {
	var count int64
	err := db.Model(&record.Repay{}).Where("owner_id = ? AND origin_record_id = ?", ownerID, originID).Count(&count).Error
	return count, err
}

func (s *Store) LinkCountsByRepay(db *gorm.DB, ownerID uint64, repayIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64, len(repayIDs))
	if len(repayIDs) == 0 {
		return out, nil
	}
	type row struct {
		RepayRecordID uint64 `gorm:"column:repay_record_id"`
		Count         int64  `gorm:"column:cnt"`
	}
	rows := make([]row, 0)
	err := db.Model(&record.Repay{}).
		Select("repay_record_id, COUNT(*) AS cnt").
		Where("owner_id = ? AND repay_record_id IN ?", ownerID, repayIDs).
		Group("repay_record_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.RepayRecordID] = r.Count
	}
	return out, nil
}

func (s *Store) OriginLinkCounts(db *gorm.DB, ownerID uint64, originIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64, len(originIDs))
	if len(originIDs) == 0 {
		return out, nil
	}
	type row struct {
		OriginRecordID uint64 `gorm:"column:origin_record_id"`
		Count          int64  `gorm:"column:cnt"`
	}
	rows := make([]row, 0)
	err := db.Model(&record.Repay{}).
		Select("origin_record_id, COUNT(*) AS cnt").
		Where("owner_id = ? AND origin_record_id IN ?", ownerID, originIDs).
		Group("origin_record_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.OriginRecordID] = r.Count
	}
	return out, nil
}

func (s *Store) DeleteLink(db *gorm.DB, ownerID uint64, repayID, originID uint64) error {
	return db.Where("owner_id = ? AND repay_record_id = ? AND origin_record_id = ?", ownerID, repayID, originID).
		Delete(&record.Repay{}).Error
}

func (s *Store) DeleteLinksByRepay(db *gorm.DB, ownerID uint64, repayID uint64) error {
	return db.Where("owner_id = ? AND repay_record_id = ?", ownerID, repayID).Delete(&record.Repay{}).Error
}

func (s *Store) SetStatus(db *gorm.DB, ownerID uint64, ids []uint64, status string) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Model(&record.Record{}).
		Where("owner_id = ? AND id IN ?", ownerID, ids).
		Update("status", status).Error
}

func (s *Store) SetPaperNoteIfEmpty(db *gorm.DB, ownerID uint64, ids []uint64, note string) error {
	if len(ids) == 0 || note == "" {
		return nil
	}
	return db.Model(&record.Record{}).
		Where("owner_id = ? AND id IN ? AND paper_note = ''", ownerID, ids).
		Update("paper_note", note).Error
}

func (s *Store) SetPaperNote(db *gorm.DB, ownerID uint64, id uint64, note string) error {
	return db.Model(&record.Record{}).Where("owner_id = ? AND id = ?", ownerID, id).
		Update("paper_note", note).Error
}

// PendingPaperNotes 返回抄录了标注但还没有转成还礼的收礼记录。
func (s *Store) PendingPaperNotes(db *gorm.DB, ownerID uint64, bookID uint64, page string) ([]record.Record, error) {
	items := make([]record.Record, 0)
	query := db.Model(&record.Record{}).
		Where("owner_id = ? AND direction = ? AND paper_note <> '' AND status <> ?", ownerID, "in", "repaid").
		Where("NOT EXISTS (SELECT 1 FROM record_repay rp WHERE rp.owner_id = ? AND rp.origin_record_id = `record`.id)", ownerID)
	if bookID > 0 {
		query = query.Where("book_id = ?", bookID)
	}
	if page != "" {
		query = query.Where("paper_page = ?", page)
	}
	if err := query.Order("occurred_on DESC, id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) PageAggs(db *gorm.DB, ownerID uint64, bookID uint64) ([]record.PageAgg, error) {
	items := make([]record.PageAgg, 0)
	err := db.Model(&record.Record{}).
		Select("paper_page, COALESCE(SUM(total_cent), 0) AS record_cent, COUNT(*) AS record_count").
		Where("owner_id = ? AND book_id = ? AND paper_page <> '' AND direction = ?", ownerID, bookID, "in").
		Group("paper_page").
		Scan(&items).Error
	return items, err
}

func (s *Store) PageChecks(db *gorm.DB, ownerID uint64, bookID uint64) ([]record.PaperPageCheck, error) {
	items := make([]record.PaperPageCheck, 0)
	err := db.Where("owner_id = ? AND book_id = ?", ownerID, bookID).Find(&items).Error
	return items, err
}

func (s *Store) UpsertPageCheck(db *gorm.DB, ownerID uint64, bookID uint64, page string, declaredCent int64) error {
	now := time.Now()
	row := &record.PaperPageCheck{
		OwnerID:      ownerID,
		BookID:       bookID,
		PaperPage:    page,
		DeclaredCent: declaredCent,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "owner_id"}, {Name: "book_id"}, {Name: "paper_page"}},
		DoUpdates: clause.Assignments(map[string]any{
			"declared_cent": declaredCent,
			"updated_at":    now,
		}),
	}).Create(row).Error
}

func (s *Store) PageRecords(db *gorm.DB, ownerID uint64, bookID uint64, page string) ([]record.Record, error) {
	items := make([]record.Record, 0)
	err := db.Where("owner_id = ? AND book_id = ? AND paper_page = ?", ownerID, bookID, page).
		Order("paper_seq ASC, occurred_on ASC, id ASC").
		Find(&items).Error
	return items, err
}

func (s *Store) CountByOwner(db *gorm.DB, ownerID uint64) (int64, error) {
	var count int64
	err := db.Model(&record.Record{}).Where("owner_id = ?", ownerID).Count(&count).Error
	return count, err
}
