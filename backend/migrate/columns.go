package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

// ColumnAddition 描述全量快照之后新增的列。新库由 schema.sql 直接建出，
// 老库的 CREATE TABLE IF NOT EXISTS 不会改表，因此由 migrate 按需补列。
type ColumnAddition struct {
	Table  string
	Column string
	DDL    string
}

// AdditiveColumns 只允许追加列；删改列仍按 README 的要求备份后重建库。
var AdditiveColumns = []ColumnAddition{
	{
		Table:  "book",
		Column: "occurred_on",
		DDL:    "ALTER TABLE book ADD COLUMN occurred_on TEXT NOT NULL DEFAULT ''",
	},
}

// EnsureColumns 给已存在的老库补齐 AdditiveColumns，可重复执行。
func EnsureColumns(tx *gorm.DB) error {
	for _, addition := range AdditiveColumns {
		if tx.Migrator().HasColumn(addition.Table, addition.Column) {
			continue
		}
		if err := tx.Exec(addition.DDL).Error; err != nil {
			return fmt.Errorf("add column %s.%s: %w", addition.Table, addition.Column, err)
		}
	}
	return nil
}
