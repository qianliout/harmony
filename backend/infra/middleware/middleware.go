// Package middleware 提供请求 ID、访问日志、登录鉴权、管理员与平台边界。
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

	// 第一阶段固定单用户；现在由登录态决定 owner。
	ownerContextKeyName = "harmony.ownerId"
)

type ownerContextKey struct{}
type tokenContextKey struct{}

// TokenVerifier 由 auth service 实现，返回令牌对应的 owner。
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (uint64, error)
}

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

// Auth 读取 Bearer 令牌，查库确认有效后把 owner 写入 ctx。
func Auth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.JsonErr(c, response.Unauthorized("请先登录"))
			c.Abort()
			return
		}
		ownerID, err := verifier.VerifyToken(c.Request.Context(), token)
		if err != nil {
			response.JsonErr(c, err)
			c.Abort()
			return
		}
		c.Set(ownerContextKeyName, ownerID)
		ctx := context.WithValue(c.Request.Context(), ownerContextKey{}, ownerID)
		ctx = context.WithValue(ctx, tokenContextKey{}, token)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AdminOnly 只允许管理员调用（识别相关接口）。
func AdminOnly(adminIDs []uint64) gin.HandlerFunc {
	allowed := make(map[uint64]bool, len(adminIDs))
	for _, id := range adminIDs {
		allowed[id] = true
	}
	return func(c *gin.Context) {
		if !allowed[OwnerID(c.Request.Context())] {
			response.JsonErr(c, response.New(403, "识别只对管理员开放", "admin only"))
			c.Abort()
			return
		}
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

// OwnerID 从请求 ctx 取 owner；取不到时返回 0，store 查不到任何数据。
func OwnerID(ctx context.Context) uint64 {
	if ctx != nil {
		if v, ok := ctx.Value(ownerContextKey{}).(uint64); ok {
			return v
		}
	}
	return 0
}

// WithOwner 供测试与内部调用构造带 owner 的 ctx。
func WithOwner(ctx context.Context, ownerID uint64) context.Context {
	return context.WithValue(ctx, ownerContextKey{}, ownerID)
}

func TokenFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(tokenContextKey{}).(string); ok {
		return v
	}
	return ""
}

type requestIDKey struct{}

func RequestIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().Format("20060102150405.000000")
	}
	return hex.EncodeToString(buf)
}
