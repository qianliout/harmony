// Package rdb 提供数据库连接与事务封装，DAL 通过 rdb.Get() 使用。
package rdb

import (
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"harmony/infra/consts"
)

// RDB 是 DAL 层唯一的数据库入口。
type RDB struct {
	db     *gorm.DB
	driver string
}

func Open(driver, dsn string) (*RDB, error) {
	dialector, err := dialectorFor(driver, dsn)
	if err != nil {
		return nil, err
	}
	silentLogger := gormlogger.Default.LogMode(gormlogger.Silent)
	config := &gorm.Config{Logger: silentLogger}
	gdb, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &RDB{db: gdb, driver: driver}, nil
}

func dialectorFor(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case consts.DBDriverPostgres:
		return postgres.Open(dsn), nil
	case consts.DBDriverMySQL:
		return mysql.Open(dsn), nil
	case consts.DBDriverSQLite:
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database driver %q", driver)
	}
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
	if err := r.db.AutoMigrate(dst...); err != nil {
		return err
	}
	return nil
}

func (r *RDB) Transaction(fn func(tx *gorm.DB) error) error {
	if err := r.db.Transaction(fn); err != nil {
		return err
	}
	return nil
}
