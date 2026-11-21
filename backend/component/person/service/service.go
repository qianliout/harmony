package service

import (
	"context"
	"fmt"

	"harmony/component/person/store"
	recordstore "harmony/component/record/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/response"
)

type PersonSrv struct {
	personDal *store.PersonDao
	recordDal *recordstore.RecordDao
	log       *logger.Logger
}

func NewPersonSrv(personDal *store.PersonDao, recordDal *recordstore.RecordDao) *PersonSrv {
	return &PersonSrv{
		personDal: personDal,
		recordDal: recordDal,
		log:       logger.New(logger.WithModule("person"), logger.WithSubModule("service")),
	}
}

func (s *PersonSrv) SearchPerson(ctx context.Context, param *person.SearchPersonParam) ([]*person.PersonView, int64, error) {
	result := make([]*person.PersonView, 0)
	rows, total, err := s.personDal.SearchPerson(ctx, param)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Msg("searchPerson failed")
		return result, 0, err
	}
	stats, err := s.statPersons(ctx, param.OwnerID, rows)
	if err != nil {
		return result, 0, err
	}
	result = s.toPersonViews(rows, stats)
	return result, total, nil
}

func (s *PersonSrv) GetPerson(ctx context.Context, param *person.GetPersonParam) (*person.PersonView, error) {
	row, err := s.getPerson(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	rows := []*person.Person{row}
	stats, err := s.statPersons(ctx, param.OwnerID, rows)
	if err != nil {
		return nil, err
	}
	views := s.toPersonViews(rows, stats)
	return views[0], nil
}

// CreatePerson 拒绝同名同备注：同名的不同的人必须靠备注区分。
func (s *PersonSrv) CreatePerson(ctx context.Context, param *person.CreatePersonParam) (*person.PersonView, error) {
	if err := s.checkDuplicate(ctx, param.OwnerID, 0, param.Name, param.Remark); err != nil {
		return nil, err
	}
	row := &person.Person{
		OwnerID: param.OwnerID,
		Name:    param.Name,
		Remark:  param.Remark,
	}
	if err := s.personDal.CreatePerson(ctx, row); err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Str("name", param.Name).Msg("createPerson failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("personID", row.ID).Msg("person created")
	view := &person.PersonView{
		ID:          row.ID,
		Name:        row.Name,
		Remark:      row.Remark,
		DisplayName: row.DisplayName(),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	return view, nil
}

// UpdatePerson 改名或改备注；姓名参与拼音派生，因此走 Serialize → Check → ToUpdater。
func (s *PersonSrv) UpdatePerson(ctx context.Context, param *person.UpdatePersonParam) (*person.PersonView, error) {
	before, err := s.getPerson(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	after := &person.Person{
		ID:      before.ID,
		OwnerID: before.OwnerID,
		Name:    param.Name,
		Remark:  param.Remark,
	}
	after = after.Serialize()
	if err := after.Check(); err != nil {
		return nil, response.NewErr(err)
	}
	getParam := &person.GetPersonParam{ID: param.ID, OwnerID: param.OwnerID}
	if before.Same(after) {
		view, err := s.GetPerson(ctx, getParam)
		if err != nil {
			return nil, err
		}
		return view, nil
	}
	if err := s.checkDuplicate(ctx, param.OwnerID, param.ID, after.Name, after.Remark); err != nil {
		return nil, err
	}
	if err := s.personDal.UpdatePerson(ctx, param.ID, after.ToUpdater()); err != nil {
		s.log.Error(ctx).Err(err).Int64("personID", param.ID).Msg("updatePerson failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("personID", param.ID).Msg("person updated")
	view, err := s.GetPerson(ctx, getParam)
	if err != nil {
		return nil, err
	}
	return view, nil
}

// DeletePerson 有关联记录时阻止删除。
func (s *PersonSrv) DeletePerson(ctx context.Context, param *person.GetPersonParam) error {
	if _, err := s.getPerson(ctx, param.OwnerID, param.ID); err != nil {
		return err
	}
	refParam := &record.SearchRecordParam{
		OwnerID:  param.OwnerID,
		PersonID: param.ID,
		Filter:   &modelfilter.Filter{Limit: 1},
	}
	_, refCount, err := s.recordDal.SearchRecord(ctx, refParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("personID", param.ID).Msg("count person records failed")
		return err
	}
	if refCount > 0 {
		zh := fmt.Sprintf("这位亲友还有 %d 条记录，先处理记录再删除", refCount)
		return response.HasReference(zh, "person has records", refCount)
	}
	if err := s.personDal.DeletePerson(ctx, param.ID); err != nil {
		s.log.Error(ctx).Err(err).Int64("personID", param.ID).Msg("deletePerson failed")
		return err
	}
	s.log.Info(ctx).Int64("personID", param.ID).Msg("person deleted")
	return nil
}

// MatchPerson 找到唯一亲友：按 id 时必须存在；按姓名时找不到返回 (nil, nil)，由调用方决定是否新建。
// 未填备注时优先取无备注的同名亲友，其次取唯一的同名亲友；仍有多个时报歧义。
func (s *PersonSrv) MatchPerson(ctx context.Context, param *person.MatchPersonParam) (*person.Person, error) {
	if param.PersonID > 0 {
		row, err := s.getPerson(ctx, param.OwnerID, param.PersonID)
		if err != nil {
			return nil, err
		}
		return row, nil
	}
	searchParam := &person.SearchPersonParam{OwnerID: param.OwnerID, Name: param.PersonName}
	rows, _, err := s.personDal.SearchPerson(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Str("name", param.PersonName).Msg("search person by name failed")
		return nil, err
	}
	for _, row := range rows {
		if row.Remark == param.PersonRemark {
			return row, nil
		}
	}
	if param.PersonRemark != "" || len(rows) == 0 {
		return nil, nil
	}
	if len(rows) == 1 {
		return rows[0], nil
	}
	zh := fmt.Sprintf("有多位叫「%s」的亲友，请选择是哪一位", param.PersonName)
	return nil, response.Conflict(consts.ReasonPersonAmbiguous, zh, "person name is ambiguous", rows)
}

func (s *PersonSrv) getPerson(ctx context.Context, ownerID, personID int64) (*person.Person, error) {
	searchParam := &person.SearchPersonParam{OwnerID: ownerID, PersonIDs: []int64{personID}}
	rows, _, err := s.personDal.SearchPerson(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("personID", personID).Msg("search person by id failed")
		return nil, err
	}
	if len(rows) == 0 {
		return nil, response.NotFound("亲友不存在", "person not found")
	}
	return rows[0], nil
}

// checkDuplicate 在同一 owner 下查找同名同备注的其他亲友；selfID 为本次更新的亲友，创建时为 0。
func (s *PersonSrv) checkDuplicate(ctx context.Context, ownerID, selfID int64, name, remark string) error {
	searchParam := &person.SearchPersonParam{OwnerID: ownerID, Name: name}
	rows, _, err := s.personDal.SearchPerson(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Str("name", name).Msg("search duplicate person failed")
		return err
	}
	for _, row := range rows {
		duplicated := row.ID != selfID && row.Remark == remark
		if !duplicated {
			continue
		}
		zh := fmt.Sprintf("已有亲友「%s」，如果是不同的人，请填写备注区分，或先给原来那位补上备注", row.DisplayName())
		return response.Conflict(consts.ReasonDuplicatePerson, zh, "duplicate person", row)
	}
	return nil
}

func (s *PersonSrv) statPersons(ctx context.Context, ownerID int64, rows []*person.Person) (map[int64]*record.RecordStat, error) {
	stats := make(map[int64]*record.RecordStat, len(rows))
	if len(rows) == 0 {
		return stats, nil
	}
	personIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		personIDs = append(personIDs, row.ID)
	}
	statParam := &record.StatRecordParam{
		OwnerID:   ownerID,
		PersonIDs: personIDs,
		GroupBy:   consts.RecordGroupByPerson,
	}
	statRows, err := s.recordDal.StatRecord(ctx, statParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", ownerID).Msg("stat person records failed")
		return stats, err
	}
	for _, stat := range statRows {
		stats[stat.GroupID] = stat
	}
	return stats, nil
}

func (s *PersonSrv) toPersonViews(rows []*person.Person, stats map[int64]*record.RecordStat) []*person.PersonView {
	views := make([]*person.PersonView, 0, len(rows))
	for _, row := range rows {
		view := &person.PersonView{
			ID:          row.ID,
			Name:        row.Name,
			Remark:      row.Remark,
			DisplayName: row.DisplayName(),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
		stat, ok := stats[row.ID]
		if ok {
			view.InCent = stat.InCent
			view.OutCent = stat.OutCent
			view.RecordCount = stat.RecordCount
			view.LastOn = stat.LastOn
		}
		views = append(views, view)
	}
	return views
}
