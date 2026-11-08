package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"harmony/component/setting/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/setting"
	"harmony/infra/rdb"
	"harmony/infra/response"
)

type SettingSrv struct {
	rdb   *rdb.RDB
	store *store.Store
	log   *logger.Logger
}

func NewSettingSrv(database *rdb.RDB, settingStore *store.Store) *SettingSrv {
	return &SettingSrv{
		rdb:   database,
		store: settingStore,
		log:   logger.New(logger.WithModule("setting"), logger.WithSubModule("service")),
	}
}

func (s *SettingSrv) Get(ctx context.Context) (*setting.Settings, error) {
	ownerID := middleware.OwnerID(ctx)
	rows, err := s.store.Get(s.rdb.Get(), ownerID)
	if err != nil {
		return nil, err
	}
	settings := setting.Default()
	if value, ok := rows[consts.SettingShowOwing]; ok {
		settings.ShowOwing = value != "false"
	}
	if value, ok := rows[consts.SettingLastBookID]; ok {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			settings.LastBookID = parsed
		}
	}
	if value, ok := rows[consts.SettingLastEventID]; ok {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			settings.LastEventID = parsed
		}
	}
	if value, ok := rows[consts.SettingQuickAmounts]; ok && strings.TrimSpace(value) != "" {
		var amounts []int64
		if err := json.Unmarshal([]byte(value), &amounts); err == nil && len(amounts) > 0 {
			settings.QuickAmounts = amounts
		}
	}
	return &settings, nil
}

func (s *SettingSrv) Update(ctx context.Context, param setting.UpdateParam) (*setting.Settings, error) {
	ownerID := middleware.OwnerID(ctx)
	if param.QuickAmounts != nil {
		if len(*param.QuickAmounts) == 0 {
			return nil, response.BadRequest("至少保留一个快捷金额", "quick amounts cannot be empty")
		}
		if len(*param.QuickAmounts) > 10 {
			return nil, response.BadRequest("快捷金额最多 10 个", "too many quick amounts")
		}
		for _, cent := range *param.QuickAmounts {
			if cent < 0 {
				return nil, response.BadRequest("快捷金额不能为负数", "quick amount must be non-negative")
			}
		}
	}
	if param.ShowOwing != nil {
		if err := s.store.Upsert(s.rdb.Get(), ownerID, consts.SettingShowOwing, boolText(*param.ShowOwing)); err != nil {
			return nil, err
		}
	}
	if param.LastBookID != nil {
		if err := s.store.Upsert(s.rdb.Get(), ownerID, consts.SettingLastBookID, strconv.FormatUint(*param.LastBookID, 10)); err != nil {
			return nil, err
		}
	}
	if param.LastEventID != nil {
		if err := s.store.Upsert(s.rdb.Get(), ownerID, consts.SettingLastEventID, strconv.FormatUint(*param.LastEventID, 10)); err != nil {
			return nil, err
		}
	}
	if param.QuickAmounts != nil {
		payload, err := json.Marshal(*param.QuickAmounts)
		if err != nil {
			return nil, err
		}
		if err := s.store.Upsert(s.rdb.Get(), ownerID, consts.SettingQuickAmounts, string(payload)); err != nil {
			return nil, err
		}
	}
	s.log.Info(ctx).Msg("setting.update")
	return s.Get(ctx)
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
