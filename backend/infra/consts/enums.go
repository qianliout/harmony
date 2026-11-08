package consts

// 业务枚举：数据库与接口统一使用短字符串，取值与 docs/01-数据库设计.md 一致。
const (
	RelationRelative = "relative"
	RelationFriend   = "friend"
	RelationColleague = "colleague"
	RelationNeighbor = "neighbor"
	RelationOther    = "other"
)

var RelationLabels = map[string]string{
	RelationRelative:  "亲戚",
	RelationFriend:    "朋友",
	RelationColleague: "同事",
	RelationNeighbor:  "邻居",
	RelationOther:     "其他",
}

const (
	BookStatusNormal = "normal"
	BookStatusMerged = "merged"
)

const (
	EventTypeWedding     = "wedding"
	EventTypeFuneral     = "funeral"
	EventTypeFullMoon    = "fullmoon"
	EventTypeHousewarming = "housewarming"
	EventTypeSchool      = "school"
	EventTypeNewYear     = "newyear"
	EventTypeOther       = "other"
)

var EventTypeLabels = map[string]string{
	EventTypeWedding:      "红事",
	EventTypeFuneral:      "白事",
	EventTypeFullMoon:     "满月",
	EventTypeHousewarming: "乔迁",
	EventTypeSchool:       "升学",
	EventTypeNewYear:      "过年",
	EventTypeOther:        "其他",
}

const (
	EventHostSelf  = "self"
	EventHostOther = "other"
)

const (
	DirectionIn  = "in"
	DirectionOut = "out"
)

var DirectionLabels = map[string]string{
	DirectionIn:  "收",
	DirectionOut: "送",
}

const (
	RecordStatusPending = "pending"
	RecordStatusRepaid  = "repaid"
	RecordStatusNone    = "none"
)

const (
	RecordSourceManual    = "manual"
	RecordSourceRecognize = "recognize"
	RecordSourceImport    = "import"
)

const (
	DatePrecisionDay   = "day"
	DatePrecisionMonth = "month"
	DatePrecisionYear  = "year"
)

const (
	DictTypeBookCategory = "book_category"
	DictTypeItemKind     = "item_kind"
)

const (
	DictCodeRenqing = "renqing"
	DictCodeHuopao  = "huopao"
	DictCodeOther   = "other"

	ItemKindCash   = "cash"
	ItemKindHuopao = "huopao"
	ItemKindGift   = "gift"
)

// 占位事项名：纸本标注没有写对方事项时使用。
const PlaceholderEventName = "纸本未注明事项"

// 识别任务状态。
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
)

// 设置项 key。
const (
	SettingShowOwing    = "show_owing"
	SettingLastBookID   = "last_book_id"
	SettingLastEventID  = "last_event_id"
	SettingQuickAmounts = "quick_amounts"
)

// DefaultQuickAmounts 是金额快捷按钮的默认值，单位分。
var DefaultQuickAmounts = []int64{10000, 20000, 50000, 60000, 80000, 100000}

// 错误 custom.reason。
const (
	ReasonNeedConfirm    = "need_confirm"
	ReasonHasReference   = "has_reference"
	ReasonStateConflict  = "state_conflict"
	ReasonPlatform       = "platform"
	ReasonPersonAmbiguous = "personName.ambiguous"
)

// 软确认 data.kind。
const (
	ConfirmDuplicatePerson = "duplicate_person"
	ConfirmDuplicateRecord = "duplicate_record"
	ConfirmUnlinkOnDelete  = "unlink_on_delete"
)

func ValidRelation(v string) bool {
	_, ok := RelationLabels[v]
	return ok
}

func ValidBookStatus(v string) bool {
	return v == BookStatusNormal || v == BookStatusMerged
}

func ValidEventType(v string) bool {
	_, ok := EventTypeLabels[v]
	return ok
}

func ValidEventHost(v string) bool {
	return v == EventHostSelf || v == EventHostOther
}

func ValidDirection(v string) bool {
	return v == DirectionIn || v == DirectionOut
}

func ValidRecordStatus(v string) bool {
	switch v {
	case RecordStatusPending, RecordStatusRepaid, RecordStatusNone:
		return true
	}
	return false
}

func ValidRecordSource(v string) bool {
	return v == RecordSourceManual || v == RecordSourceRecognize || v == RecordSourceImport
}

func ValidDatePrecision(v string) bool {
	return v == DatePrecisionDay || v == DatePrecisionMonth || v == DatePrecisionYear
}

func ValidDictType(v string) bool {
	return v == DictTypeBookCategory || v == DictTypeItemKind
}

// DictSeed 是内置字典项的定义；schema.sql 的 owner=1 种子与新建用户的初始化共用这一份。
type DictSeed struct {
	DictType string
	Code     string
	Label    string
	Color    string
	Sort     int
}

var BuiltinDictItems = []DictSeed{
	{DictType: DictTypeBookCategory, Code: DictCodeRenqing, Label: "人情礼", Color: "cinnabar", Sort: 10},
	{DictType: DictTypeBookCategory, Code: DictCodeHuopao, Label: "火炮", Color: "amber", Sort: 20},
	{DictType: DictTypeBookCategory, Code: DictCodeOther, Label: "其他", Color: "stone", Sort: 30},
	{DictType: DictTypeItemKind, Code: ItemKindCash, Label: "现金", Color: "cinnabar", Sort: 10},
	{DictType: DictTypeItemKind, Code: ItemKindHuopao, Label: "火炮", Color: "amber", Sort: 20},
	{DictType: DictTypeItemKind, Code: ItemKindGift, Label: "实物礼品", Color: "teal", Sort: 30},
}
