package store

import (

	"gorm.io/gorm"

	"harmony/infra/model/event"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/record"
	"harmony/infra/utils"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Get(db *gorm.DB, ownerID uint64, id uint64) (*event.Event, error) {
	var item event.Event
	if err := db.Where("owner_id = ? AND id = ?", ownerID, id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]event.Event, error) {
	items := make([]event.Event, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := db.Where("owner_id = ? AND id IN ?", ownerID, ids).Find(&items).Error
	return items, err
}

func (s *Store) List(db *gorm.DB, ownerID uint64, q event.Query, f *modelfilter.Filter) ([]event.Event, int64, error) {
	query := db.Model(&event.Event{}).Where("owner_id = ?", ownerID)
	if !q.IncludePlaceholder {
		query = query.Where("placeholder = 0")
	}
	if q.Type != "" {
		query = query.Where("type = ?", q.Type)
	}
	if q.Host != "" {
		query = query.Where("host = ?", q.Host)
	}
	if q.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+q.Keyword+"%")
	}
	if from, err := utils.ParseDate(q.From); err == nil {
		query = query.Where("event_date >= ?", utils.DateString(from))
	}
	if to, err := utils.ParseDate(q.To); err == nil {
		query = query.Where("event_date <= ?", utils.DateString(to))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]event.Event, 0)
	query = query.Order("event_date IS NULL ASC, event_date DESC, id DESC")
	if f != nil {
		query = query.Offset(int(f.Offset)).Limit(int(f.Limit))
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Recent 返回最近使用的事项，供选事项器使用。
func (s *Store) Recent(db *gorm.DB, ownerID uint64, limit int) ([]event.Event, error) {
	if limit <= 0 {
		limit = 6
	}
	ids := make([]uint64, 0)
	err := db.Model(&record.Record{}).
		Select("event_id").
		Where("owner_id = ?", ownerID).
		Group("event_id").
		Order("MAX(created_at) DESC").
		Limit(limit).
		Pluck("event_id", &ids).Error
	if err != nil || len(ids) == 0 {
		return []event.Event{}, err
	}
	items, err := s.GetMany(db, ownerID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint64]event.Event, len(items))
	for i := range items {
		byID[items[i].ID] = items[i]
	}
	ordered := make([]event.Event, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok && !item.Placeholder {
			ordered = append(ordered, item)
		}
	}
	return ordered, nil
}

func (s *Store) Create(db *gorm.DB, item *event.Event) error {
	if item.Type == "" {
		item.Type = "other"
	}
	if item.Host == "" {
		item.Host = "other"
	}
	if item.DatePrecision == "" {
		item.DatePrecision = "day"
	}
	return db.Create(item).Error
}

func (s *Store) Update(db *gorm.DB, item *event.Event) error {
	return db.Model(item).Select("name", "type", "event_date", "location", "host", "host_person_id", "date_precision", "remark").Updates(item).Error
}

func (s *Store) Delete(db *gorm.DB, ownerID uint64, id uint64) error {
	return db.Where("owner_id = ? AND id = ?", ownerID, id).Delete(&event.Event{}).Error
}

func (s *Store) CountRecords(db *gorm.DB, ownerID uint64, eventID uint64) (int64, error) {
	var count int64
	err := db.Model(&record.Record{}).Where("owner_id = ? AND event_id = ?", ownerID, eventID).Count(&count).Error
	return count, err
}

// FindByName 用于导入时按名称匹配事项。
func (s *Store) FindByName(db *gorm.DB, ownerID uint64, name string) (*event.Event, error) {
	var item event.Event
	if err := db.Where("owner_id = ? AND name = ?", ownerID, name).Order("id ASC").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// FindByHostAndName 用于还礼时复用"这个人 + 事项名"的已有事项。
func (s *Store) FindByHostAndName(db *gorm.DB, ownerID uint64, hostPersonID uint64, name string) (*event.Event, error) {
	var item event.Event
	query := db.Where("owner_id = ? AND name = ? AND host_person_id = ?", ownerID, name, hostPersonID)
	if err := query.Order("id ASC").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) Placeholder(db *gorm.DB, ownerID uint64) (*event.Event, error) {
	var item event.Event
	if err := db.Where("owner_id = ? AND placeholder = 1", ownerID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

type aggRow struct {
	EventID      uint64 `gorm:"column:event_id"`
	RecordCount  int64  `gorm:"column:record_count"`
	InCent       int64  `gorm:"column:in_cent"`
	OutCent      int64  `gorm:"column:out_cent"`
	PersonCount  int64  `gorm:"column:person_count"`
	RepaidCount  int64  `gorm:"column:repaid_count"`
	PendingCount int64  `gorm:"column:pending_count"`
}

func (s *Store) Aggregates(db *gorm.DB, ownerID uint64, eventIDs []uint64) (map[uint64]event.Summary, error) {
	out := make(map[uint64]event.Summary, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	rows := make([]aggRow, 0)
	err := db.Model(&record.Record{}).
		Select(`event_id,
			COUNT(*) AS record_count,
			COALESCE(SUM(CASE WHEN direction = 'in' THEN total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN direction = 'out' THEN total_cent ELSE 0 END), 0) AS out_cent,
			COUNT(DISTINCT person_id) AS person_count,
			COALESCE(SUM(CASE WHEN direction = 'in' AND status = 'repaid' THEN 1 ELSE 0 END), 0) AS repaid_count,
			COALESCE(SUM(CASE WHEN direction = 'in' AND status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_count`).
		Where("owner_id = ? AND event_id IN ?", ownerID, eventIDs).
		Group("event_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.EventID] = event.Summary{
			RecordCount:  row.RecordCount,
			InCent:       row.InCent,
			OutCent:      row.OutCent,
			PersonCount:  row.PersonCount,
			RepaidCount:  row.RepaidCount,
			PendingCount: row.PendingCount,
		}
	}
	return out, nil
}

