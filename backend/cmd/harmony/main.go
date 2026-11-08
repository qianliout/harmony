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
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	authapi "harmony/component/auth/api"
	authservice "harmony/component/auth/service"
	authstore "harmony/component/auth/store"
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
	"harmony/migrate"
	"harmony/web"
	"gorm.io/gorm"
)

// version 由构建参数注入：-ldflags "-X main.version=$(git describe --tags --always)"。
var version = consts.ServiceVersion

func main() {
	// SetDefault 必须先于任何构造执行：每个 struct 都从默认 logger 构建自己的 logger，更早构建的会沿用旧输出。
	if err := logger.SetDefault(logger.WithConsole()); err != nil {
		fmt.Fprintf(os.Stderr, "set default logger failed: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(logger.WithModule(consts.ServiceName))

	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		command := os.Args[1]
		args := os.Args[2:]
		switch command {
		case "migrate":
			if err := runMigrate(args, log); err != nil {
				log.Error(context.Background()).Err(err).Msg("migrate failed")
				os.Exit(1)
			}
			return
		case "backup":
			if err := runBackup(args, log); err != nil {
				log.Error(context.Background()).Err(err).Msg("backup failed")
				os.Exit(1)
			}
			return
		case "token":
			if err := runToken(args, log); err != nil {
				log.Error(context.Background()).Err(err).Msg("token command failed")
				os.Exit(1)
			}
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q; available: migrate, backup, token\n", command)
			os.Exit(2)
		}
	}

	fs := flag.NewFlagSet("harmony", flag.ExitOnError)
	configPath := fs.String("config", "", "YAML config file path; falls back to defaults and env vars")
	_ = fs.Parse(os.Args[1:])
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

func loadConfig(args []string) error {
	fs := flag.NewFlagSet("harmony", flag.ExitOnError)
	path := fs.String("config", "", "YAML config file path")
	_ = fs.Parse(args)
	return config.Load(*path)
}

// runMigrate 执行内嵌的 SQLite 全量快照；快照使用 IF NOT EXISTS 与 ON CONFLICT，可重复执行。
func runMigrate(args []string, log *logger.Logger) error {
	if err := loadConfig(args); err != nil {
		return err
	}
	db, err := rdb.Open(config.Global.Database.Path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, statement := range migrate.Statements() {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("execute schema statement: %w", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	mode, _ := db.JournalMode(context.Background())
	log.Info(context.Background()).Str("path", config.Global.Database.Path).Str("journalMode", mode).Msg("migrate done")
	return nil
}

// runToken issue 在服务器上签发一个网页访问令牌，供小程序审核前登录网页使用。
func runToken(args []string, log *logger.Logger) error {
	if len(args) == 0 || args[0] != "issue" {
		return fmt.Errorf("usage: harmony token issue -config <path> -user 1 -days 14")
	}
	fs := flag.NewFlagSet("token issue", flag.ExitOnError)
	configPath := fs.String("config", "", "YAML config file path")
	userID := fs.Uint64("user", 1, "user id")
	days := fs.Int64("days", 14, "token valid days")
	_ = fs.Parse(args[1:])
	if err := config.Load(*configPath); err != nil {
		return err
	}
	db, err := rdb.Open(config.Global.Database.Path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	authSrv, err := authservice.NewAuthSrv(db, authstore.New(), nil, *config.Global)
	if err != nil {
		return err
	}
	login, err := authSrv.IssueTokenForCLI(context.Background(), *userID, *days)
	if err != nil {
		return err
	}
	log.Info(context.Background()).Uint64("userId", *userID).Msg("token issued")
	fmt.Printf("token: %s\nexpiresAt: %d\nuser: %s\n", login.Token, login.ExpiresAt, login.User.Nickname)
	return nil
}

// runBackup 用 VACUUM INTO 生成在线备份，并清理过期令牌与旧备份。
func runBackup(args []string, log *logger.Logger) error {
	if err := loadConfig(args); err != nil {
		return err
	}
	cfg := config.Global
	db, err := rdb.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := os.MkdirAll(cfg.Backup.Dir, 0o750); err != nil {
		return err
	}
	stamp := time.Now().Format("20060102-1504")
	target := filepath.Join(cfg.Backup.Dir, "harmony-"+stamp+".db")
	escaped := strings.ReplaceAll(target, "'", "''")
	if err := db.Get().Exec("VACUUM INTO '" + escaped + "'").Error; err != nil {
		return fmt.Errorf("vacuum into backup: %w", err)
	}
	if err := db.Get().Exec("DELETE FROM auth_token WHERE expires_at < ?", time.Now().Add(-7*24*time.Hour)).Error; err != nil {
		log.Warn(context.Background()).Err(err).Msg("prune expired tokens failed")
	}
	if cfg.Backup.KeepDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -cfg.Backup.KeepDays)
		entries, err := os.ReadDir(cfg.Backup.Dir)
		if err == nil {
			for _, entry := range entries {
				info, infoErr := entry.Info()
				if infoErr != nil || entry.IsDir() || !strings.HasPrefix(entry.Name(), "harmony-") {
					continue
				}
				if info.ModTime().Before(cutoff) {
					_ = os.Remove(filepath.Join(cfg.Backup.Dir, entry.Name()))
				}
			}
		}
	}
	log.Info(context.Background()).Str("path", target).Msg("backup done")
	return nil
}

// run 装配依赖、注册路由并提供 HTTP 服务，直到 ctx 结束。
func run(ctx context.Context, log *logger.Logger, cfg *config.Config) error {
	db, err := rdb.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	journalMode, modeErr := db.JournalMode(ctx)
	if modeErr != nil {
		log.Warn(ctx).Err(modeErr).Msg("check journal mode failed")
	} else {
		log.Info(ctx).Str("path", cfg.Database.Path).Str("journalMode", journalMode).Msg("database opened")
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

	authStore := authstore.New()
	wechatClient := authservice.NewWechatClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret)
	authSrv, err := authservice.NewAuthSrv(db, authStore, wechatClient, *cfg)
	if err != nil {
		return err
	}

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
	recognizeSrv := recognizeservice.NewRecognizeSrv(db, taskStore, recordSrv, personStore, recordStore, authSrv, recognizer, cfg.Recognize, cfg.Storage.UploadDir)
	if err := recognizeSrv.FailStale(ctx); err != nil {
		log.Warn(ctx).Err(err).Msg("mark stale recognize tasks failed")
	}
	transferSrv := transferservice.NewTransferSrv(db, recordSrv, personStore, eventStore, bookStore, dictStore)

	healthAPI := healthapi.NewHealthAPI(healthSrv)
	authAPI := authapi.NewAuthAPI(authSrv)
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

	v1 := engine.Group("/api/v1", middleware.RequestID(), middleware.AccessLog())
	healthapi.RegisterRoutes(v1, healthAPI)
	authapi.RegisterPublicRoutes(v1, authAPI)
	recognizeapi.RegisterPublicRoutes(v1, recognizeAPI)

	protected := v1.Group("", middleware.Auth(authSrv))
	authapi.RegisterRoutes(protected, authAPI)
	dictapi.RegisterRoutes(protected, dictAPI)
	personapi.RegisterRoutes(protected, personAPI)
	bookapi.RegisterRoutes(protected, bookAPI)
	eventapi.RegisterRoutes(protected, eventAPI)
	recordapi.RegisterRoutes(protected, recordAPI)
	viewapi.RegisterRoutes(protected, viewAPI)
	statsapi.RegisterRoutes(protected, statsAPI)
	searchapi.RegisterRoutes(protected, searchAPI)
	settingapi.RegisterRoutes(protected, settingAPI)
	transferapi.RegisterRoutes(protected, transferAPI)
	recognizeGroup := protected.Group("", middleware.PlatformGuard(), middleware.AdminOnly(cfg.Auth.AdminUserIDs))
	recognizeapi.RegisterRoutes(recognizeGroup, recognizeAPI)

	if err := os.MkdirAll(cfg.Storage.UploadDir, 0o755); err != nil {
		return fmt.Errorf("create upload dir: %w", err)
	}
	registerH5(engine)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
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
