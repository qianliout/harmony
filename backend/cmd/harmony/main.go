// Package main 是 harmony 服务入口：为 Web 端与后续小程序提供 /api/v1 REST 接口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"harmony/component/config"
	healthapi "harmony/component/health/api"
	healthservice "harmony/component/health/service"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/rdb"
)

func main() {
	configPath := flag.String("config", "", "YAML config file path; falls back to defaults and env vars")
	flag.Parse()

	// SetDefault 必须先于任何构造执行：每个 struct 都从默认 logger 构建自己的 logger，更早构建的会沿用旧输出。
	if err := logger.SetDefault(logger.WithConsole()); err != nil {
		fmt.Fprintf(os.Stderr, "set default logger failed: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(logger.WithModule(consts.ServiceName))

	if err := config.Load(*configPath); err != nil {
		log.Error(context.Background()).Err(err).Msg("load config failed")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, log, config.Global); err != nil {
		log.Error(ctx).Err(err).Msg("run server failed")
		os.Exit(1)
	}
}

// run 装配依赖、注册路由并提供 HTTP 服务，直到 ctx 结束。
func run(ctx context.Context, log *logger.Logger, cfg *config.Config) error {
	db, err := rdb.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Error(ctx).Err(closeErr).Msg("close database failed")
		}
	}()

	healthSrv := healthservice.NewHealthSrv(db)
	healthAPI := healthapi.NewHealthAPI(healthSrv)

	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	v1 := engine.Group("/api/v1")
	healthapi.RegisterRoutes(v1, healthAPI)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: engine}
	log.Info(ctx).Str("addr", addr).Msg("harmony server listening")

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		serveFailed := err != nil && !errors.Is(err, http.ErrServerClosed)
		if serveFailed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	}
}
