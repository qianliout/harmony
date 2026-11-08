package consts

// 日志固定字段名与日志文件权限。
const (
	LogFieldModule    = "module"
	LogFieldSubModule = "subModule"

	LogDirPerm  = 0755
	LogFilePerm = 0644
)

// 统一响应信封的默认文案与语言协商。
const (
	RespMsgSuccess      = "success"
	RespMsgSystemError  = "系统错误"
	RespMsgUnknownError = "未知错误"

	RespErrWithItems = "WithItems expected array or slice"
	RespErrWithItem  = "WithItem expected struct, struct pointer or map"

	HeaderAcceptLanguage = "Accept-Language"
	LangEN               = "en"
)

// 列表过滤：排序方向、排序列与查询参数名。
const (
	FilterSortDesc = "desc"
	FilterSortAsc  = "asc"

	FilterSortFieldID        = "id"
	FilterSortFieldCreatedAt = "created_at"

	FilterQueryOffset    = "offset"
	FilterQueryLimit     = "limit"
	FilterQuerySortBy    = "sort_by"
	FilterQuerySortField = "sort_field"

	// FilterDefaultLimit 是 SetDefault 的默认页大小。
	FilterDefaultLimit int64 = 10
	// FilterNilLimit 是 AddFilterWithDefault 收到 nil filter 时的兜底条数。
	FilterNilLimit int64 = 200
)

// 分页查询参数名（小驼峰形态）与页大小上限。
const (
	QueryPage      = "page"
	QueryPageSize  = "pageSize"
	QuerySortBy    = "sortBy"
	QuerySortField = "sortField"

	// QueryListSep 是查询参数里列表值的分隔符。
	QueryListSep = ","

	DefaultPageSize int64 = 20
	MaxPageSize     int64 = 100
)

// LLM 输出里包裹 JSON 的 Markdown 代码围栏。
const (
	LLMFence     = "```"
	LLMFenceLang = "json"

	EmptyJSONArray = "[]"
)
