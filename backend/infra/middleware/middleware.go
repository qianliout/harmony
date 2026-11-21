// Package middleware 提供请求 ID、访问日志与固定单用户上下文。
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"harmony/infra/consts"
	"harmony/infra/logger"
)

const HeaderRequestID = "X-Request-Id"

type ownerContextKey struct{}

type requestIDKey struct{}

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

// Owner 把固定单用户写入请求上下文，本机阶段不做登录。
func Owner() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := WithOwner(c.Request.Context(), consts.LocalOwnerID)
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
			Int64("ownerId", OwnerID(c.Request.Context())).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int64("latencyMs", time.Since(start).Milliseconds()).
			Msg("request")
	}
}

// OwnerID 从请求 ctx 取 owner；取不到时返回 0，由 param.Check 拒绝。
func OwnerID(ctx context.Context) int64 {
	if v, ok := ctx.Value(ownerContextKey{}).(int64); ok {
		return v
	}
	return 0
}

// WithOwner 供中间件与测试构造带 owner 的 ctx。
func WithOwner(ctx context.Context, ownerID int64) context.Context {
	return context.WithValue(ctx, ownerContextKey{}, ownerID)
}

func RequestIDFrom(ctx context.Context) string {
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
