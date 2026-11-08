package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"harmony/component/dict/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/dict"
	"harmony/infra/response"
	"harmony/infra/rdb"
	"harmony/infra/utils"
)

type DictSrv struct {
	rdb   *rdb.RDB
	store *store.Store
	log   *logger.Logger
}

func NewDictSrv(database *rdb.RDB, dictStore *store.Store) *DictSrv {
	return &DictSrv{
		rdb:   database,
		store: dictStore,
		log:   logger.New(logger.WithModule("dict"), logger.WithSubModule("service")),
	}
}

func (s *DictSrv) List(ctx context.Context, dictType string) ([]dict.Resp, error) {
	if dictType != "" && !consts.ValidDictType(dictType) {
		return nil, response.BadRequest("字典类型不正确", "invalid dict type")
	}
	ownerID := middleware.OwnerID(ctx)
	items, err := s.store.List(s.rdb.Get(), ownerID, dictType)
	if err != nil {
		return nil, err
	}
	out := make([]dict.Resp, 0, len(items))
	for i := range items {
		out = append(out, items[i].Serialize())
	}
	return out, nil
}

func (s *DictSrv) Create(ctx context.Context, param dict.CreateParam) (*dict.Resp, error) {
	param.Type = strings.TrimSpace(param.Type)
	param.Label = strings.TrimSpace(param.Label)
	if !consts.ValidDictType(param.Type) {
		return nil, response.BadRequest("字典类型不正确", "invalid dict type")
	}
	if param.Label == "" {
		return nil, response.BadRequest("名称不能为空", "label is required")
	}
	ownerID := middleware.OwnerID(ctx)
	item := &dict.Item{
		OwnerID:  ownerID,
		DictType: param.Type,
		Code:     utils.RandomCode("c_"),
		Label:    param.Label,
		Color:    normalizeColor(param.Color),
		Builtin:  false,
	}
	if err := s.store.Create(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	resp := item.Serialize()
	s.log.Info(ctx).Uint64("id", item.ID).Str("type", item.DictType).Msg("dict.create")
	return &resp, nil
}

func (s *DictSrv) Update(ctx context.Context, id uint64, param dict.UpdateParam) (*dict.Resp, error) {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("字典项不存在")
		}
		return nil, err
	}
	label := strings.TrimSpace(param.Label)
	if label == "" {
		return nil, response.BadRequest("名称不能为空", "label is required")
	}
	item.Label = label
	if param.Color != "" {
		item.Color = normalizeColor(param.Color)
	}
	if param.Sort != nil {
		item.Sort = *param.Sort
	}
	if err := s.store.Update(s.rdb.Get(), item); err != nil {
		return nil, err
	}
	resp := item.Serialize()
	return &resp, nil
}

func (s *DictSrv) Delete(ctx context.Context, id uint64) error {
	ownerID := middleware.OwnerID(ctx)
	item, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("字典项不存在")
		}
		return err
	}
	if item.Builtin {
		return response.BadRequest("内置项不能删除", "builtin dict item cannot be deleted")
	}
	refs, err := s.countRefs(ownerID, item)
	if err != nil {
		return err
	}
	if refs > 0 {
		return response.HasReference("已经有记录在引用这个字典项", refs)
	}
	if err := s.store.Delete(s.rdb.Get(), ownerID, id); err != nil {
		return err
	}
	s.log.Info(ctx).Uint64("id", id).Msg("dict.delete")
	return nil
}

func (s *DictSrv) countRefs(ownerID uint64, item *dict.Item) (int64, error) {
	switch item.DictType {
	case consts.DictTypeBookCategory:
		return s.store.CountBookCategoryRefs(s.rdb.Get(), ownerID, item.Code)
	case consts.DictTypeItemKind:
		return s.store.CountItemKindRefs(s.rdb.Get(), ownerID, item.Code)
	}
	return 0, nil
}

func normalizeColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "stone"
	}
	return color
}
