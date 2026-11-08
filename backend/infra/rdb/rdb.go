// Package rdb 提供 SQLite 连接与事务封装，DAL 通过 rdb.Get() 使用。
package rdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const driverName = "sqlite"

// RDB 是 DAL 层唯一的数据库入口。
type RDB struct {
	db          *gorm.DB
	driver      string
	path        string
	journalMode string
}

// Open 打开 SQLite 文件，pragma 固定拼接，不让配置文件改动。
func Open(path string) (*RDB, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create database dir: %w", err)
		}
	}
	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(0)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_txlock=immediate"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("database handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(0)

	var journalMode string
	if err := gdb.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		return nil, fmt.Errorf("check journal mode: %w", err)
	}
	return &RDB{db: gdb, driver: driverName, path: path, journalMode: journalMode}, nil
}

// New 用已有连接构造 RDB；测试传入事务句柄后，嵌套的 Transaction 走 savepoint，随外层回滚。
func New(db *gorm.DB, driver string) *RDB {
	return &RDB{db: db, driver: driver}
}

func (r *RDB) Get() *gorm.DB {
	return r.db
}

func (r *RDB) Driver() string {
	return r.driver
}

func (r *RDB) Path() string {
	return r.path
}

// JournalMode 返回启动时 PRAGMA journal_mode 的结果，健康检查使用。
func (r *RDB) JournalMode(ctx context.Context) (string, error) {
	if r.journalMode != "" {
		return r.journalMode, nil
	}
	var mode string
	if err := r.db.WithContext(ctx).Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		return "", err
	}
	return mode, nil
}

func (r *RDB) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (r *RDB) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (r *RDB) AutoMigrate(dst ...any) error {
	return r.db.AutoMigrate(dst...)
}

func (r *RDB) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
