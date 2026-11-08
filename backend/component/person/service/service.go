package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"harmony/component/person/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type PersonSrv struct {
	rdb   *rdb.RDB
	store *store.Store
	log   *logger.Logger
}

func NewPersonSrv(database *rdb.RDB, personStore *store.Store) *PersonSrv {
	return &PersonSrv{
		rdb:   database,
		store: personStore,
		log:   logger.New(logger.WithModule("person"), logger.WithSubModule("service")),
	}
}

func (s *PersonSrv) List(ctx context.Context, q person.Query, f *modelfilter.Filter) ([]person.Brief, int64, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Relation != "" && !consts.ValidRelation(q.Relation) {
		return nil, 0, response.BadRequest("关系取值不正确", "invalid relation")
	}
	ownerID := middleware.OwnerID(ctx)
	items, total, err := s.store.List(s.rdb.Get(), ownerID, q, f)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	summaries, err := s.store.Summaries(s.rdb.Get(), ownerID, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]person.Brief, 0, len(items))
	for i := range items {
		summary := summaries[items[i].ID]
		out = append(out, person.Brief{
			ID:            items[i].ID,
			Name:          items[i].Name,
			Relation:      items[i].Relation,
			RelationLabel: person.LabelOf(items[i].Relation),
			Phone:         items[i].Phone,
			Remark:        items[i].Remark,
			CreatedAt:     utils.Millis(items[i].CreatedAt),
			UpdatedAt:     utils.Millis(items[i].UpdatedAt),
			InCent:        summary.InCent,
			OutCent:       summary.OutCent,
			PendingCount:  summary.PendingCount,
			LastDate:      summary.LastDate,
		})
	}
	return out, total, nil
}

func (s *PersonSrv) Recent(ctx context.Context, limit int) ([]person.Brief, error) {
	ownerID := middleware.OwnerID(ctx)
	items, err := s.store.Recent(s.rdb.Get(), ownerID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]person.Brief, 0, len(items))
	for i := range items {
		out = append(out, person.Brief{
			ID:            items[i].ID,
			Name:          items[i].Name,
			Relation:      items[i].Relation,
			RelationLabel: person.LabelOf(items[i].Relation),
			Phone:         items[i].Phone,
			Remark:        items[i].Remark,
			CreatedAt:     utils.Millis(items[i].CreatedAt),
			UpdatedAt:     utils.Millis(items[i].UpdatedAt),
		})
	}
	return out, nil
}

func (s *PersonSrv) Detail(ctx context.Context, id uint64) (*person.Detail, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("亲友不存在")
		}
		return nil, err
	}
	summaries, err := s.store.Summaries(s.rdb.Get(), ownerID, []uint64{id})
	if err != nil {
		return nil, err
	}
	return &person.Detail{
		ID:            item.ID,
		Name:          item.Name,
		Relation:      item.Relation,
		RelationLabel: person.LabelOf(item.Relation),
		Phone:         item.Phone,
		Remark:        item.Remark,
		CreatedAt:     utils.Millis(item.CreatedAt),
		UpdatedAt:     utils.Millis(item.UpdatedAt),
		Summary:       summaries[id],
	}, nil
}

func (s *PersonSrv) Create(ctx context.Context, param person.CreateParam) (*person.Detail, error) {
	name := strings.TrimSpace(param.Name)
	relation := strings.TrimSpace(param.Relation)
	if name == "" {
		return nil, response.BadRequest("姓名不能为空", "name is required")
	}
	if len([]rune(name)) > 32 {
		return nil, response.BadRequest("姓名不能超过 32 个字", "name is too long")
	}
	if relation == "" {
		relation = consts.RelationOther
	}
	if !consts.ValidRelation(relation) {
		return nil, response.BadRequest("关系取值不正确", "invalid relation")
	}
	ownerID := middleware.OwnerID(ctx)
	duplicates, err := s.store.FindExactByName(s.rdb.Get(), ownerID, name)
	if err != nil {
		return nil, err
	}
	if len(duplicates) > 0 && !param.Confirm {
		conflicts := make([]response.ConfirmConflict, 0, len(duplicates))
		for i := range duplicates {
			conflicts = append(conflicts, response.ConfirmConflict{
				ID:        duplicates[i].ID,
				Summary:   duplicates[i].Name + " · " + person.LabelOf(duplicates[i].Relation),
				CreatedAt: utils.Millis(duplicates[i].CreatedAt),
			})
		}
		payload := response.ConfirmPayload{
			Kind:        consts.ConfirmDuplicatePerson,
			Title:       "已存在同名亲友",
			Message:     "已经有 " + name + " 了，仍然要新建吗？",
			Conflicts:   conflicts,
			ConfirmText: "仍然新建",
		}
		return nil, response.Conflict(consts.ReasonNeedConfirm, payload.Message, payload)
	}

	full, initials := utils.Pinyin(name)
	item := &person.Person{
		OwnerID:      ownerID,
		Name:         name,
		NamePinyin:   full,
		NameInitials: initials,
		Relation:     relation,
		Phone:        strings.TrimSpace(param.Phone),
		Remark:       strings.TrimSpace(param.Remark),
	}
	if err := s.store.Create(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("id", item.ID).Msg("person.create")
	return &person.Detail{
		ID:            item.ID,
		Name:          item.Name,
		Relation:      item.Relation,
		RelationLabel: person.LabelOf(item.Relation),
		Phone:         item.Phone,
		Remark:        item.Remark,
		CreatedAt:     utils.Millis(item.CreatedAt),
		UpdatedAt:     utils.Millis(item.UpdatedAt),
		Summary:       person.Summary{},
	}, nil
}

func (s *PersonSrv) Update(ctx context.Context, id uint64, param person.UpdateParam) (*person.Detail, error) {
	name := strings.TrimSpace(param.Name)
	relation := strings.TrimSpace(param.Relation)
	if name == "" {
		return nil, response.BadRequest("姓名不能为空", "name is required")
	}
	if relation == "" {
		relation = consts.RelationOther
	}
	if !consts.ValidRelation(relation) {
		return nil, response.BadRequest("关系取值不正确", "invalid relation")
	}
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("亲友不存在")
		}
		return nil, err
	}
	duplicates, err := s.store.FindExactByName(s.rdb.Get(), ownerID, name)
	if err != nil {
		return nil, err
	}
	otherSameName := make([]person.Person, 0)
	for i := range duplicates {
		if duplicates[i].ID != id {
			otherSameName = append(otherSameName, duplicates[i])
		}
	}
	if len(otherSameName) > 0 && !param.Confirm {
		conflicts := make([]response.ConfirmConflict, 0, len(otherSameName))
		for i := range otherSameName {
			conflicts = append(conflicts, response.ConfirmConflict{
				ID:        otherSameName[i].ID,
				Summary:   otherSameName[i].Name + " · " + person.LabelOf(otherSameName[i].Relation),
				CreatedAt: utils.Millis(otherSameName[i].CreatedAt),
			})
		}
		payload := response.ConfirmPayload{
			Kind:        consts.ConfirmDuplicatePerson,
			Title:       "已存在同名亲友",
			Message:     "已经有 " + name + " 了，仍然要改名吗？",
			Conflicts:   conflicts,
			ConfirmText: "仍然改名",
		}
		return nil, response.Conflict(consts.ReasonNeedConfirm, payload.Message, payload)
	}

	full, initials := utils.Pinyin(name)
	item.Name = name
	item.NamePinyin = full
	item.NameInitials = initials
	item.Relation = relation
	item.Phone = strings.TrimSpace(param.Phone)
	item.Remark = strings.TrimSpace(param.Remark)
	if err := s.store.Update(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("person.update")
	return s.Detail(ctx, id)
}

func (s *PersonSrv) Delete(ctx context.Context, id uint64) error {
	ownerID := middleware.OwnerID(ctx)
	if _, err := s.store.Get(s.rdb.Get(), ownerID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("亲友不存在")
		}
		return err
	}
	count, err := s.store.CountRecords(s.rdb.Get(), ownerID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return response.HasReference("这位亲友还有 "+strconv.FormatInt(count, 10)+" 条记录，先处理记录再删除", count)
	}
	if err := s.store.Delete(s.rdb.Get(), ownerID, id); err != nil {
		return err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("person.delete")
	return nil
}

