# Logger 使用示例

业务层 struct 持有自己的 logger，并在构造函数里固定 `module` 和 `subModule`。

```go
package image

import (
	"context"
	"time"

	"harmony/infra/logger"
)

type ImageSrv struct {
	log *logger.Logger
}

func NewImageSrv() *ImageSrv {
	return &ImageSrv{
		log: logger.New(
			logger.WithConsole(),
			logger.WithModule("image"),
			logger.WithSubModule("service"),
		),
	}
}

func (s *ImageSrv) Search(ctx context.Context, uniqueID int64) error {
	s.log.Info(ctx).
		Int64("uniqueID", uniqueID).
		Msg("search image start")

	if err := s.search(ctx, uniqueID); err != nil {
		s.log.Error(ctx).
			Err(err).
			Int64("uniqueID", uniqueID).
			Msg("search image failed")
		return err
	}

	return nil
}

func (s *ImageSrv) search(ctx context.Context, uniqueID int64) error {
	return nil
}
```

日志默认输出到命令行。需要写入文件时，使用 `NewWithError` 显式处理文件打开错误。

```go
log, err := logger.NewWithError(
	logger.WithFile("logs/app.log"),
	logger.WithModule("image"),
	logger.WithSubModule("service"),
)
if err != nil {
	return nil, err
}
```

同时输出到命令行和文件：

```go
log, err := logger.NewWithError(
	logger.WithConsoleAndFile("logs/app.log"),
	logger.WithModule("image"),
	logger.WithSubModule("service"),
)
if err != nil {
	return nil, err
}
```

临时切换模块信息时，显式从当前 logger 派生，不把日志字段写入 `context.Context`。

```go
log := logger.New(logger.WithModule("image"), logger.WithSubModule("service"))
log.WithSubModule("api").
	Info(ctx).
	Int64("uniqueID", uniqueID).
	Msg("create image")
```

临时 debug 日志默认不输出。排查问题时显式打开，几分钟后自动失效。

```go
log := logger.New(
	logger.WithModule("image"),
	logger.WithSubModule("service"),
)

log.DebugFor(5 * time.Minute)
log.Debug(ctx).
	Int64("uniqueID", uniqueID).
	Str("param", param.LogStr()).
	Msg("search image debug")
```

也可以按全局、模块或子模块开启临时 debug。

```go
logger.EnableDebugFor(3 * time.Minute)
logger.EnableModuleDebugFor("image", 5*time.Minute)
logger.EnableSubModuleDebugFor("image", "service", 5*time.Minute)
```
