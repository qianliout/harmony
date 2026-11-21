package response

import (
	"errors"
	"net/http"

	"harmony/infra/consts"
)

// 错误构造函数：传输层恒为 HTTP 200，业务状态放在 code，前端按 code 加 custom.reason 分支处理。

func New(code int64, zh, en string) *ErrI18 {
	return &ErrI18{Code: code, Msg: zh, Err: errors.New(en), EN: en, ZH: zh}
}

func BadRequest(zh, en string) *ErrI18 {
	return New(http.StatusBadRequest, zh, en)
}

func NotFound(zh, en string) *ErrI18 {
	return New(http.StatusNotFound, zh, en)
}

func Forbidden(zh, en string) *ErrI18 {
	return New(http.StatusForbidden, zh, en)
}

func BadGateway(zh, en string) *ErrI18 {
	return New(http.StatusBadGateway, zh, en)
}

// Conflict 构造 409；reason 写入 custom.reason，data 携带结构化明细。
func Conflict(reason, zh, en string, data any) *ErrI18 {
	err := New(http.StatusConflict, zh, en)
	err.Data = data
	err.Custom = map[string]any{"reason": reason}
	return err
}

func StateConflict(zh, en string) *ErrI18 {
	return Conflict(consts.ReasonStateConflict, zh, en, nil)
}

// HasReference 表示删除对象仍有关联记录。
func HasReference(zh, en string, refCount int64) *ErrI18 {
	return Conflict(consts.ReasonHasReference, zh, en, map[string]any{"refCount": refCount})
}
