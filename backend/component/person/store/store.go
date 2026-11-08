package store

import (

	"gorm.io/gorm"

	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) Get(db *gorm.DB, ownerID uint64, id uint64) (*person.Person, error) {
	var p person.Person
	if err := db.Where("owner_id = ? AND id = ?", ownerID, id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetMany(db *gorm.DB, ownerID uint64, ids []uint64) ([]person.Person, error) {
	items := make([]person.Person, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := db.Where("owner_id = ? AND id IN ?", ownerID, ids).Find(&items).Error
	return items, err
}

func (s *Store) List(db *gorm.DB, ownerID uint64, q person.Query, f *modelfilter.Filter) ([]person.Person, int64, error) {
	query := db.Model(&person.Person{}).Where("owner_id = ?", ownerID)
	if q.Relation != "" {
		query = query.Where("relation = ?", q.Relation)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		prefix := q.Keyword + "%"
		query = query.Where("(name LIKE ? OR name_pinyin LIKE ? OR name_initials LIKE ?)", like, prefix, prefix)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]person.Person, 0)
	if f != nil {
		query = modelfilter.AddFilter(query, f)
	} else {
		query = query.Order("updated_at DESC, id DESC")
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindExactByName 精确匹配姓名，用于记账与还礼时的选人规则。
func (s *Store) FindExactByName(db *gorm.DB, ownerID uint64, name string) ([]person.Person, error) {
	items := make([]person.Person, 0)
	err := db.Where("owner_id = ? AND name = ?", ownerID, name).
		Order("id ASC").Find(&items).Error
	return items, err
}

// Recent 返回最近有记账往来的人，供选人器使用。
func (s *Store) Recent(db *gorm.DB, ownerID uint64, limit int) ([]person.Person, error) {
	if limit <= 0 {
		limit = 8
	}
	ids := make([]uint64, 0)
	err := db.Model(&record.Record{}).
		Select("person_id").
		Where("owner_id = ?", ownerID).
		Group("person_id").
		Order("MAX(created_at) DESC").
		Limit(limit).
		Pluck("person_id", &ids).Error
	if err != nil || len(ids) == 0 {
		return []person.Person{}, err
	}
	items := make([]person.Person, 0)
	if err := db.Where("owner_id = ? AND id IN ?", ownerID, ids).Find(&items).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint64]person.Person, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	ordered := make([]person.Person, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			ordered = append(ordered, item)
		}
	}
	return ordered, nil
}

func (s *Store) Create(db *gorm.DB, p *person.Person) error {
	return db.Create(p).Error
}

func (s *Store) Update(db *gorm.DB, p *person.Person) error {
	return db.Model(p).Select("name", "name_pinyin", "name_initials", "relation", "phone", "remark").Updates(p).Error
}

func (s *Store) Delete(db *gorm.DB, ownerID uint64, id uint64) error {
	return db.Where("owner_id = ? AND id = ?", ownerID, id).Delete(&person.Person{}).Error
}

func (s *Store) CountRecords(db *gorm.DB, ownerID uint64, personID uint64) (int64, error) {
	var count int64
	err := db.Model(&record.Record{}).Where("owner_id = ? AND person_id = ?", ownerID, personID).Count(&count).Error
	return count, err
}

func (s *Store) CountAll(db *gorm.DB, ownerID uint64) (int64, error) {
	var count int64
	err := db.Model(&person.Person{}).Where("owner_id = ?", ownerID).Count(&count).Error
	return count, err
}

type summaryRow struct {
	PersonID     uint64    `gorm:"column:person_id"`
	InCent       int64     `gorm:"column:in_cent"`
	OutCent      int64     `gorm:"column:out_cent"`
	PendingCount int64     `gorm:"column:pending_count"`
	PendingCent  int64     `gorm:"column:pending_cent"`
	RecordCount  int64     `gorm:"column:record_count"`
	LastDate     string `gorm:"column:last_date"`
}

// Summaries 一条 GROUP BY 查询批量取得多个人的汇总，避免逐人查询。
func (s *Store) Summaries(db *gorm.DB, ownerID uint64, personIDs []uint64) (map[uint64]person.Summary, error) {
	out := make(map[uint64]person.Summary, len(personIDs))
	if len(personIDs) == 0 {
		return out, nil
	}
	rows := make([]summaryRow, 0)
	err := db.Model(&record.Record{}).
		Select(`person_id,
			COALESCE(SUM(CASE WHEN direction = 'in' THEN total_cent ELSE 0 END), 0) AS in_cent,
			COALESCE(SUM(CASE WHEN direction = 'out' THEN total_cent ELSE 0 END), 0) AS out_cent,
			COALESCE(SUM(CASE WHEN direction = 'in' AND status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_count,
			COALESCE(SUM(CASE WHEN direction = 'in' AND status = 'pending' THEN total_cent ELSE 0 END), 0) AS pending_cent,
			COUNT(*) AS record_count,
			MAX(occurred_on) AS last_date`).
		Where("owner_id = ? AND person_id IN ?", ownerID, personIDs).
		Group("person_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PersonID] = person.Summary{
			InCent:       row.InCent,
			OutCent:      row.OutCent,
			NetCent:      row.InCent - row.OutCent,
			PendingCount: row.PendingCount,
			PendingCent:  row.PendingCent,
			RecordCount:  row.RecordCount,
			LastDate:     row.LastDate,
		}
	}
	return out, nil
}
