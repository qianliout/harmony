// Package middleware 提供请求 ID、访问日志、owner 注入与平台边界。
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"harmony/infra/logger"
	"harmony/infra/response"
)

const (
	HeaderRequestID = "X-Request-Id"
	HeaderPlatform  = "X-Client-Platform"
	PlatformH5      = "h5"

	// 第一阶段固定单用户；第二阶段改为从 Authorization 解析。
	DefaultOwnerID uint64 = 1
)

type ownerContextKey struct{}

// RequestID 读取或生成 X-Request-Id，写入响应头与请求 ctx。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Header(HeaderRequestID, requestID)
		ctx := context.WithValue(c.Request.Context(), requestIDKey{}, requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AccessLog 每个请求一行访问日志。
func AccessLog() gin.HandlerFunc {
	log := logger.New(logger.WithModule("http"), logger.WithSubModule("access"))
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info(c.Request.Context()).
			Str("requestId", RequestIDFrom(c.Request.Context())).
			Uint64("ownerId", OwnerID(c.Request.Context())).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int64("latencyMs", time.Since(start).Milliseconds()).
			Msg("request")
	}
}

// Owner 第一阶段把 ownerID 固定为 1 注入 ctx；第二阶段换成 token 解析。
func Owner() gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerID := DefaultOwnerID
		c.Set(ownerContextKeyName, ownerID)
		ctx := context.WithValue(c.Request.Context(), ownerContextKey{}, ownerID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// PlatformGuard 只挂在识别路由上：小程序端不得调用识别接口。
func PlatformGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := strings.ToLower(strings.TrimSpace(c.GetHeader(HeaderPlatform)))
		if platform != PlatformH5 {
			response.JsonErr(c, response.Platform("识别功能仅在网页版开放", "recognition is only available on web"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// OwnerID 从请求 ctx 取 owner；第一阶段未注入时按单用户处理。
func OwnerID(ctx context.Context) uint64 {
	if ctx != nil {
		if v, ok := ctx.Value(ownerContextKey{}).(uint64); ok && v > 0 {
			return v
		}
	}
	return DefaultOwnerID
}

type requestIDKey struct{}

const ownerContextKeyName = "harmony.ownerId"

func RequestIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().Format("20060102150405.000000")
	}
	return hex.EncodeToString(buf)
}
