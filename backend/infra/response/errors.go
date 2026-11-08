package response

import (
	"errors"
	"net/http"
)

// 错误构造函数：沿用仓库约定，传输层恒为 HTTP 200，业务状态放在 code。
// 前端按 code 加 custom.reason 分支处理，详见 docs/02-后端设计.md。

func New(code int64, zh, en string) *ErrI18 {
	return &ErrI18{Code: code, Msg: zh, Err: errors.New(zh), EN: en, ZH: zh}
}

func BadRequest(zh, en string) *ErrI18 {
	return New(http.StatusBadRequest, zh, en)
}

// BadRequestDetail 用于需要细分原因的 400，例如 personName.ambiguous。
func BadRequestDetail(errCode, zh string, custom map[string]any) *ErrI18 {
	err := BadRequest(zh, errCode)
	err.Err = errors.New(errCode)
	err.Custom = custom
	return err
}

func NotFound(zh string) *ErrI18 {
	return New(http.StatusNotFound, zh, zh)
}

func Unauthorized(zh string) *ErrI18 {
	return New(http.StatusUnauthorized, zh, zh)
}

// Conflict 构造 409；data 为确认信息，reason 写入 custom.reason。
func Conflict(reason, zh string, data any) *ErrI18 {
	err := New(http.StatusConflict, zh, zh)
	err.Data = data
	err.Custom = map[string]any{"reason": reason}
	return err
}

func StateConflict(zh string) *ErrI18 {
	return Conflict("state_conflict", zh, nil)
}

func HasReference(zh string, refCount int64) *ErrI18 {
	return Conflict("has_reference", zh, map[string]any{"refCount": refCount})
}

func Platform(zh, en string) *ErrI18 {
	err := New(http.StatusForbidden, zh, en)
	err.Custom = map[string]any{"reason": "platform"}
	return err
}

func BadGateway(zh string) *ErrI18 {
	return New(http.StatusBadGateway, zh, zh)
}

// ConfirmConflict 是软确认弹层里的冲突明细。
type ConfirmConflict struct {
	ID        uint64 `json:"id"`
	Summary   string `json:"summary"`
	CreatedAt int64  `json:"createdAt"`
}

// ConfirmPayload 对应 docs/02 的软确认协议 data 字段。
type ConfirmPayload struct {
	Kind        string            `json:"kind"`
	Title       string            `json:"title"`
	Message     string            `json:"message"`
	Conflicts   []ConfirmConflict `json:"conflicts"`
	ConfirmText string            `json:"confirmText"`
}
