package response

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"

	"harmony/infra/consts"
)

// Body 是 HTTP 层统一的响应信封。
type Body struct {
	Code       int64          `json:"code"`
	Msg        string         `json:"msg"`
	Err        string         `json:"err"`
	Data       any            `json:"data"` // 列表响应必须是非 nil 切片
	Total      int64          `json:"totalItems"`
	PerPage    int64          `json:"itemsPerPage"`
	StartIndex int64          `json:"startIndex"`
	Custom     map[string]any `json:"custom"`
}

// ErrI18 携带中英双语文案的错误；Data/Custom 用于软确认与关联引用等结构化信息。
type ErrI18 struct {
	Code   int64
	Msg    string
	Err    error
	EN     string
	ZH     string
	Data   any
	Custom map[string]any
}

func (e *ErrI18) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Msg
}

func (e *ErrI18) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewErr 把请求解析或校验失败包装成 400；已是 *ErrI18 时原样返回。
func NewErr(err error) *ErrI18 {
	var errI18 *ErrI18
	isI18 := errors.As(err, &errI18)
	if isI18 {
		return errI18
	}
	if err == nil {
		err = errors.New(consts.RespMsgBadRequest)
	}
	return &ErrI18{
		Code: int64(http.StatusBadRequest),
		Msg:  err.Error(),
		Err:  err,
		EN:   err.Error(),
		ZH:   err.Error(),
	}
}

type Option func(o *Body)

func JSONOK(c *gin.Context, opts ...Option) {
	body := Body{
		Code: int64(http.StatusOK),
		Msg:  consts.RespMsgSuccess,
	}
	for _, opt := range opts {
		opt(&body)
	}
	c.JSON(http.StatusOK, &body)
}

func JSONError(c *gin.Context, err error) {
	body := Body{
		Code: int64(http.StatusInternalServerError),
		Msg:  consts.RespMsgSystemError,
		Err:  "",
	}
	if err != nil {
		body.Err = err.Error()
	}

	var errI18 *ErrI18
	isI18 := errors.As(err, &errI18)
	if isI18 {
		body.Code = errI18.Code
		body.Msg = errMsg(c, errI18)
		body.Err = errI18.Error()
		if errI18.Data != nil {
			body.Data = errI18.Data
		}
		body.Custom = errI18.Custom
	}
	c.JSON(http.StatusOK, &body)
}

func errMsg(c *gin.Context, errI18 *ErrI18) string {
	// 收到 typed nil 的 *ErrI18 时 errors.As 也会命中，这里兜底。
	if errI18 == nil {
		return consts.RespMsgUnknownError
	}

	lang := strings.ToLower(c.GetHeader(consts.HeaderAcceptLanguage))
	if lang == consts.LangEN {
		return errI18.EN
	}
	return errI18.ZH
}

func WithItems(items any) Option {
	val := reflect.ValueOf(items)
	kind := val.Kind()
	isList := kind == reflect.Array || kind == reflect.Slice
	if !isList {
		return func(ev *Body) {
			ev.Err = consts.RespErrWithItems
		}
	}

	return func(ev *Body) {
		if val.Len() == 0 {
			ev.Data = []any{}
			return
		}
		ev.Data = items
	}
}

func WithItem(item any) Option {
	val := reflect.ValueOf(item)
	kind := val.Kind()
	if kind == reflect.Invalid {
		return func(ev *Body) {
			ev.Err = consts.RespErrWithItem
		}
	}

	itemType := val.Type()
	isStructPtr := kind == reflect.Ptr && itemType.Elem().Kind() == reflect.Struct
	isAccepted := kind == reflect.Struct || kind == reflect.Map || isStructPtr
	if !isAccepted {
		return func(ev *Body) {
			ev.Err = consts.RespErrWithItem
		}
	}

	return func(ev *Body) {
		ev.Data = item
	}
}

func WithTotalItems(n int64) Option {
	return func(ev *Body) {
		ev.Total = n
	}
}

func WithItemsPerPage(n int64) Option {
	return func(ev *Body) {
		ev.PerPage = n
	}
}

func WithStartIndex(n int64) Option {
	return func(ev *Body) {
		ev.StartIndex = n
	}
}
