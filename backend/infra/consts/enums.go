package consts

// 二元状态字段的取值，字段类型为 string。
const (
	TrueString  = "true"
	FalseString = "false"
)

// LocalOwnerID 是本机单用户固定归属，所有请求按该 owner 处理。
const LocalOwnerID int64 = 1

// 方向：收（别人来我家）/ 送（我去别人家）。
const (
	DirectionIn  = "in"
	DirectionOut = "out"
)

func ValidDirection(v string) bool {
	return v == DirectionIn || v == DirectionOut
}

// 礼簿种类：预置三类，前端允许输入自定义值。
const (
	BookCategoryRenqing = "renqing"
	BookCategoryHuopao  = "huopao"
	BookCategoryOther   = "other"
)

var BookCategoryLabels = map[string]string{
	BookCategoryRenqing: "人情礼",
	BookCategoryHuopao:  "火炮",
	BookCategoryOther:   "其他",
}

func CategoryLabel(code string) string {
	if label, ok := BookCategoryLabels[code]; ok {
		return label
	}
	return code
}

// 字段长度上限，按字符计。
const (
	PersonNameMaxLen     = 32
	PersonRemarkMaxLen   = 32
	BookNameMaxLen       = 64
	BookCategoryMaxLen   = 32
	RecordMatterMaxLen   = 128
	RecordNoteMaxLen     = 256
	RecognizeErrorMaxLen = 500
)

// PersonDisplaySep 连接姓名与备注，展示为「张三 · 村东」。
const PersonDisplaySep = " · "

// 记录统计的分组列。
const (
	RecordGroupByNone   = ""
	RecordGroupByPerson = "person_id"
	RecordGroupByBook   = "book_id"
)

// 各资源允许的排序列，未列出的列一律拒绝。
var (
	PersonSortFields = map[string]bool{"": true, "id": true, "name": true, "created_at": true, "updated_at": true}
	BookSortFields   = map[string]bool{"": true, "id": true, "name": true, "created_at": true, "updated_at": true}
	RecordSortFields = map[string]bool{"": true, "id": true, "occurred_on": true, "amount_cent": true, "created_at": true}
)

// 识别任务状态。
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
)

// 识别出的姓名与已有亲友的匹配结果。
const (
	MatchStatusNone     = "none"
	MatchStatusSingle   = "single"
	MatchStatusMultiple = "multiple"
)

// 识别图片约束。
const (
	RecognizeImageMaxSize int64 = 5 << 20
	RecognizeConcurrency        = 2

	MIMEJPEG = "image/jpeg"
	MIMEPNG  = "image/png"
)

var RecognizeImageExt = map[string]string{
	MIMEJPEG: "jpg",
	MIMEPNG:  "png",
}

// 错误 custom.reason。
const (
	ReasonHasReference    = "has_reference"
	ReasonStateConflict   = "state_conflict"
	ReasonDuplicatePerson = "duplicate_person"
	ReasonPersonAmbiguous = "person_ambiguous"
)
