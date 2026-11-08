// Package migrate 内嵌 SQLite 全量建表快照，供 migrate 子命令与测试使用。
package migrate

import (
	_ "embed"
	"strings"
)

//go:embed sqlite/schema.sql
var schema string

// Schema 返回完整建表快照。
func Schema() string {
	return schema
}

// Statements 按分号切分快照；快照里没有存储过程，不需要更复杂的解析。
func Statements() []string {
	parts := strings.Split(schema, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
