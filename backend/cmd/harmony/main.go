// Package main 是 harmony 服务入口：为 Web 端与后续小程序提供 /api/v1 REST 接口，并托管 H5 构建产物。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	bookapi "harmony/component/book/api"
	bookservice "harmony/component/book/service"
	bookstore "harmony/component/book/store"
	"harmony/component/config"
	dictapi "harmony/component/dict/api"
	dictservice "harmony/component/dict/service"
	dictstore "harmony/component/dict/store"
	eventapi "harmony/component/event/api"
	eventservice "harmony/component/event/service"
	eventstore "harmony/component/event/store"
	healthapi "harmony/component/health/api"
	healthservice "harmony/component/health/service"
	personapi "harmony/component/person/api"
	personservice "harmony/component/person/service"
	personstore "harmony/component/person/store"
	recognizeapi "harmony/component/recognize/api"
	recognizeservice "harmony/component/recognize/service"
	recognizestore "harmony/component/recognize/store"
	recordapi "harmony/component/record/api"
	recordservice "harmony/component/record/service"
	recordstore "harmony/component/record/store"
	searchapi "harmony/component/search/api"
	searchservice "harmony/component/search/service"
	searchstore "harmony/component/search/store"
	settingapi "harmony/component/setting/api"
	settingservice "harmony/component/setting/service"
	settingstore "harmony/component/setting/store"
	statsapi "harmony/component/stats/api"
	statsservice "harmony/component/stats/service"
	statsstore "harmony/component/stats/store"
	transferapi "harmony/component/transfer/api"
	transferservice "harmony/component/transfer/service"
	viewapi "harmony/component/view/api"
	viewservice "harmony/component/view/service"
	viewstore "harmony/component/view/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/rdb"
	"harmony/web"
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

	dictStore := dictstore.New()
	personStore := personstore.New()
	bookStore := bookstore.New()
	eventStore := eventstore.New()
	recordStore := recordstore.New()
	viewStore := viewstore.New()
	statsStore := statsstore.New()
	searchStore := searchstore.New()
	settingStore := settingstore.New()
	taskStore := recognizestore.New()

	healthSrv := healthservice.NewHealthSrv(db)
	dictSrv := dictservice.NewDictSrv(db, dictStore)
	personSrv := personservice.NewPersonSrv(db, personStore)
	bookSrv := bookservice.NewBookSrv(db, bookStore, dictStore, eventStore)
	eventSrv := eventservice.NewEventSrv(db, eventStore, personStore)
	recordSrv := recordservice.NewRecordSrv(db, recordStore, personStore, eventStore, bookStore, dictStore)
	settingSrv := settingservice.NewSettingSrv(db, settingStore)
	viewSrv := viewservice.NewViewSrv(db, viewStore, recordStore, recordSrv, personStore, eventStore, settingStore)
	statsSrv := statsservice.NewStatsSrv(db, statsStore, viewSrv, personStore, bookStore, dictStore)
	searchSrv := searchservice.NewSearchSrv(db, searchStore, recordSrv)
	recognizer := recognizeservice.NewOpenAICompatRecognizer(cfg.Recognize)
	recognizeSrv := recognizeservice.NewRecognizeSrv(db, taskStore, recordSrv, personStore, recordStore, recognizer, cfg.Recognize, cfg.Storage.UploadDir)
	if err := recognizeSrv.FailStale(ctx); err != nil {
		log.Warn(ctx).Err(err).Msg("mark stale recognize tasks failed")
	}
	transferSrv := transferservice.NewTransferSrv(db, recordSrv, personStore, eventStore, bookStore, dictStore)

	healthAPI := healthapi.NewHealthAPI(healthSrv)
	dictAPI := dictapi.NewDictAPI(dictSrv)
	personAPI := personapi.NewPersonAPI(personSrv)
	bookAPI := bookapi.NewBookAPI(bookSrv)
	eventAPI := eventapi.NewEventAPI(eventSrv)
	recordAPI := recordapi.NewRecordAPI(recordSrv)
	viewAPI := viewapi.NewViewAPI(viewSrv)
	statsAPI := statsapi.NewStatsAPI(statsSrv)
	searchAPI := searchapi.NewSearchAPI(searchSrv)
	settingAPI := settingapi.NewSettingAPI(settingSrv)
	recognizeAPI := recognizeapi.NewRecognizeAPI(recognizeSrv)
	transferAPI := transferapi.NewTransferAPI(transferSrv)

	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.MaxMultipartMemory = 8 << 20

	v1 := engine.Group("/api/v1", middleware.RequestID(), middleware.AccessLog(), middleware.Owner())
	healthapi.RegisterRoutes(v1, healthAPI)
	dictapi.RegisterRoutes(v1, dictAPI)
	personapi.RegisterRoutes(v1, personAPI)
	bookapi.RegisterRoutes(v1, bookAPI)
	eventapi.RegisterRoutes(v1, eventAPI)
	recordapi.RegisterRoutes(v1, recordAPI)
	viewapi.RegisterRoutes(v1, viewAPI)
	statsapi.RegisterRoutes(v1, statsAPI)
	searchapi.RegisterRoutes(v1, searchAPI)
	settingapi.RegisterRoutes(v1, settingAPI)
	transferapi.RegisterRoutes(v1, transferAPI)
	recognizeGroup := v1.Group("", middleware.PlatformGuard())
	recognizeapi.RegisterRoutes(recognizeGroup, recognizeAPI)

	if err := os.MkdirAll(cfg.Storage.UploadDir, 0o755); err != nil {
		return fmt.Errorf("create upload dir: %w", err)
	}
	engine.Static("/uploads", cfg.Storage.UploadDir)
	registerH5(engine)

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

// registerH5 托管 make release 复制到 web/dist 的 H5 产物；未构建时只提供 API。
func registerH5(engine *gin.Engine) {
	dist, err := fs.Sub(web.Files, "dist")
	if err != nil {
		return
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		engine.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "harmony API is running. Run `make release` to bundle the H5 build, or use `pnpm dev:h5`.")
		})
		return
	}
	fileServer := http.FileServer(http.FS(dist))
	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/uploads/") {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "msg": "接口不存在"})
			return
		}
		// H5 使用 hash 路由，静态服务器只需要按文件路径返回资源。
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
