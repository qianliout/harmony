package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"harmony/infra/consts"
)

var (
	defaultLogger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	nopLogger     = zerolog.Nop()
	debugGuard    = temporaryDebug{
		items: make(map[string]time.Time),
	}
)

// Logger 是项目统一日志入口；业务 struct 各自持有，并在构造时固定 module/subModule。
type Logger struct {
	base      zerolog.Logger
	log       zerolog.Logger
	module    string
	subModule string
	attrs     []any
	err       error
}

type Option func(*Logger)

type temporaryDebug struct {
	mu    sync.RWMutex
	items map[string]time.Time
}

func New(opts ...Option) *Logger {
	vi := Logger{
		base: defaultLogger,
	}
	for _, opt := range opts {
		opt(&vi)
	}
	vi.rebuild()
	return &vi
}

func NewWithError(opts ...Option) (*Logger, error) {
	vi := New(opts...)
	return vi, vi.err
}

// SetDefault 替换之后所有 New 继承的输出，必须在任何 New 之前调用：已构造的 Logger 保持原输出。
// 这里只有输出类 Option 生效，module/subModule 仍由持有 Logger 的 struct 设置。
func SetDefault(opts ...Option) error {
	vi := New(opts...)
	if vi.err != nil {
		return vi.err
	}
	defaultLogger = vi.base
	return nil
}

func WithModule(module string) Option {
	return func(vi *Logger) {
		vi.module = strings.TrimSpace(module)
	}
}

func WithSubModule(subModule string) Option {
	return func(vi *Logger) {
		vi.subModule = strings.TrimSpace(subModule)
	}
}

func WithOutput(w io.Writer) Option {
	return func(vi *Logger) {
		if w == nil {
			return
		}
		vi.setOutput(w)
	}
}

func WithConsole() Option {
	return WithOutput(os.Stdout)
}

func WithFile(path string) Option {
	return func(vi *Logger) {
		w, err := openFile(path)
		if err != nil {
			vi.err = err
			return
		}
		vi.setOutput(w)
	}
}

func WithConsoleAndFile(path string) Option {
	return func(vi *Logger) {
		w, err := openFile(path)
		if err != nil {
			vi.err = err
			return
		}
		vi.setOutput(zerolog.MultiLevelWriter(os.Stdout, w))
	}
}

func WithAttrs(args ...any) Option {
	return func(vi *Logger) {
		vi.attrs = append(vi.attrs, args...)
	}
}

func (vi *Logger) WithModule(module string) *Logger {
	cp := vi.clone()
	cp.module = strings.TrimSpace(module)
	cp.rebuild()
	return cp
}

func (vi *Logger) WithSubModule(subModule string) *Logger {
	cp := vi.clone()
	cp.subModule = strings.TrimSpace(subModule)
	cp.rebuild()
	return cp
}

func (vi *Logger) With(args ...any) *Logger {
	cp := vi.clone()
	cp.attrs = append(cp.attrs, args...)
	cp.rebuild()
	return cp
}

func (vi *Logger) Zerolog() zerolog.Logger {
	return *vi.current()
}

func (vi *Logger) Debug(ctx context.Context) *zerolog.Event {
	enabled := vi.DebugEnabled()
	if !enabled {
		event := nopLogger.Debug().Ctx(ctx)
		return event
	}
	log := vi.current()
	event := log.Debug().Ctx(ctx)
	return event
}

func (vi *Logger) Info(ctx context.Context) *zerolog.Event {
	log := vi.current()
	event := log.Info().Ctx(ctx)
	return event
}

func (vi *Logger) Warn(ctx context.Context) *zerolog.Event {
	log := vi.current()
	event := log.Warn().Ctx(ctx)
	return event
}

func (vi *Logger) Error(ctx context.Context) *zerolog.Event {
	log := vi.current()
	event := log.Error().Ctx(ctx)
	return event
}

func (vi *Logger) DebugFor(ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	debugGuard.enable(vi.debugKey(), ttl)
}

func (vi *Logger) DebugEnabled() bool {
	globalEnabled := debugGuard.enabled("")
	moduleEnabled := debugGuard.enabled(moduleDebugKey(vi.module, ""))
	subModuleEnabled := debugGuard.enabled(vi.debugKey())
	return globalEnabled || moduleEnabled || subModuleEnabled
}

func EnableDebugFor(ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	debugGuard.enable("", ttl)
}

func EnableModuleDebugFor(module string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	debugGuard.enable(moduleDebugKey(module, ""), ttl)
}

func EnableSubModuleDebugFor(module string, subModule string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	debugGuard.enable(moduleDebugKey(module, subModule), ttl)
}

func (vi *Logger) clone() *Logger {
	if vi == nil {
		return New()
	}
	cp := *vi
	cp.attrs = append([]any(nil), vi.attrs...)
	return &cp
}

func (vi *Logger) current() *zerolog.Logger {
	if vi == nil {
		return &defaultLogger
	}
	return &vi.log
}

func (vi *Logger) rebuild() {
	ctx := addContextFields(vi.base.With(), vi.attrs...)
	if vi.module != "" {
		ctx = ctx.Str(consts.LogFieldModule, vi.module)
	}
	if vi.subModule != "" {
		ctx = ctx.Str(consts.LogFieldSubModule, vi.subModule)
	}
	vi.log = ctx.Logger()
}

func (vi *Logger) setOutput(w io.Writer) {
	vi.base = zerolog.New(w).With().Timestamp().Logger()
}

func (vi *Logger) debugKey() string {
	if vi == nil {
		return ""
	}
	return moduleDebugKey(vi.module, vi.subModule)
}

func moduleDebugKey(module string, subModule string) string {
	module = strings.TrimSpace(module)
	subModule = strings.TrimSpace(subModule)
	if module == "" && subModule == "" {
		return ""
	}
	if subModule == "" {
		return module
	}
	return module + "/" + subModule
}

// addContextFields 必须使用并返回每次构建后的 ctx：zerolog.Context 是值类型，丢弃返回值会静默丢掉全部字段，
// module/subModule 曾因此从所有日志行消失。
func addContextFields(ctx zerolog.Context, args ...any) zerolog.Context {
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		invalidKey := !ok || strings.TrimSpace(key) == ""
		if invalidKey {
			continue
		}
		if i+1 >= len(args) {
			ctx = ctx.Interface(key, nil)
			continue
		}
		ctx = ctx.Interface(key, args[i+1])
	}
	return ctx
}

func openFile(path string) (*os.File, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("logger file path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, consts.LogDirPerm); err != nil {
		return nil, fmt.Errorf("create logger dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, consts.LogFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open logger file: %w", err)
	}
	return file, nil
}

func (vi *temporaryDebug) enable(key string, ttl time.Duration) {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.items[key] = time.Now().Add(ttl)
}

func (vi *temporaryDebug) enabled(key string) bool {
	vi.mu.RLock()
	until, ok := vi.items[key]
	vi.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().Before(until) {
		return true
	}
	vi.mu.Lock()
	latest, ok := vi.items[key]
	// 加写锁前可能已被其他 goroutine 续期，复查后才删除。
	expired := ok && !time.Now().Before(latest)
	if expired {
		delete(vi.items, key)
	}
	vi.mu.Unlock()
	return false
}
