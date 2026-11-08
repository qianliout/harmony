// Package consts 是 harmony 项目级常量。
package consts

// 服务标识，健康检查与日志共用。
const (
	ServiceName    = "harmony"
	ServiceVersion = "v0.1.0"
)

// 健康检查结论。
const HealthStatusOK = "ok"

// Gin 运行模式，取值与 gin.SetMode 接受的值一致。
const (
	ServerModeDebug   = "debug"
	ServerModeRelease = "release"
	ServerModeTest    = "test"
)

func ValidServerMode(mode string) bool {
	switch mode {
	case ServerModeDebug, ServerModeRelease, ServerModeTest:
		return true
	}
	return false
}


// TimestampMilliThreshold 区分秒级与毫秒级时间戳：低于它的正值是统一毫秒前存下的秒级值。
const TimestampMilliThreshold int64 = 1_000_000_000_000
