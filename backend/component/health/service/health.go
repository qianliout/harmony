package service

import (
	"context"
	"fmt"
	"time"

	"harmony/infra/consts"
	"harmony/infra/logger"
	healthmodel "harmony/infra/model/health"
)

// databasePinger 是健康检查对数据库的最小依赖，由 *rdb.RDB 满足。
type databasePinger interface {
	Ping(ctx context.Context) error
	Driver() string
	JournalMode(ctx context.Context) (string, error)
}

type HealthSrv struct {
	db  databasePinger
	log *logger.Logger
}

func NewHealthSrv(db databasePinger) *HealthSrv {
	return &HealthSrv{
		db:  db,
		log: logger.New(logger.WithModule("health"), logger.WithSubModule("service")),
	}
}

func (s *HealthSrv) Check(ctx context.Context) (*healthmodel.Status, error) {
	journalMode, err := s.db.JournalMode(ctx)
	if err != nil {
		s.log.Error(ctx).Err(err).Msg("check journal mode failed")
		return nil, fmt.Errorf("journal mode check failed: %w", err)
	}
	status := &healthmodel.Status{
		Status:      consts.HealthStatusOK,
		Database:    s.db.Driver(),
		JournalMode: journalMode,
		Version:     consts.ServiceVersion,
		Time:        time.Now().UTC().UnixMilli(),
	}
	if err := s.db.Ping(ctx); err != nil {
		s.log.Error(ctx).Err(err).Msg("ping database failed")
		return nil, fmt.Errorf("database ping failed: %w", err)
	}
	return status, nil
}
