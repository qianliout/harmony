// Package recognize 定义识别任务、识别项与批量写入入参。
package recognize

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"harmony/infra/consts"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/utils"
)

// RecognizeCandidate 是同名亲友候选。
type RecognizeCandidate struct {
	PersonID     int64  `json:"personId"`
	PersonName   string `json:"personName"`
	PersonRemark string `json:"personRemark"`
}

// RecognizeMatch 表示识别出的姓名与已有亲友的匹配情况，Status 取 consts.MatchStatus*。
type RecognizeMatch struct {
	Status     string               `json:"status"`
	PersonID   int64                `json:"personId"`
	Candidates []RecognizeCandidate `json:"candidates"`
}

// RecognizeItem 是识别出的一行；Confidence 为 0~100 的百分比。
type RecognizeItem struct {
	Seq        int64          `json:"seq"`
	Name       string         `json:"name"`
	AmountCent int64          `json:"amountCent"`
	Note       string         `json:"note"`
	Raw        string         `json:"raw"`
	Confidence int64          `json:"confidence"`
	Match      RecognizeMatch `json:"match"`
}

// UniqueID: not generated because 同一张图片允许重复上传识别，每次都是独立任务。
type RecognizeTask struct {
	ID int64 `gorm:"primaryKey;column:id" json:"id"`

	OwnerID      int64           `gorm:"column:owner_id" json:"ownerId"`
	ImagePath    string          `gorm:"column:image_path" json:"-"`
	ImageSHA256  string          `gorm:"column:image_sha256" json:"imageSha256"`
	ImageSize    int64           `gorm:"column:image_size" json:"imageSize"`
	Status       string          `gorm:"column:status" json:"status"`
	Provider     string          `gorm:"column:provider" json:"provider"`
	Model        string          `gorm:"column:model" json:"model"`
	RawResponse  string          `gorm:"column:raw_response" json:"-"`
	Items        []RecognizeItem `gorm:"-" json:"items"`
	ItemsJSON    string          `gorm:"column:items" json:"-"`
	SumCent      int64           `gorm:"column:sum_cent" json:"sumCent"`
	ErrorMessage string          `gorm:"column:error_message" json:"errorMessage"`
	Committed    string          `gorm:"column:committed" json:"committed"`
	FinishedAt   int64           `gorm:"column:finished_at" json:"finishedAt"`

	CreatedAt int64 `gorm:"autoCreateTime:milli;column:created_at" json:"createdAt"`
	UpdatedAt int64 `gorm:"autoUpdateTime:milli;column:updated_at" json:"updatedAt"`
}

func (vi *RecognizeTask) TableName() string { return "recognize_task" }

// Serialize 把识别项序列化到文本列，并派生合计金额。
func (vi *RecognizeTask) Serialize() *RecognizeTask {
	if vi == nil {
		return &RecognizeTask{Items: []RecognizeItem{}}
	}
	vi.Status = strings.TrimSpace(vi.Status)
	vi.ErrorMessage = strings.TrimSpace(vi.ErrorMessage)
	if vi.Status == "" {
		vi.Status = consts.TaskStatusPending
	}
	if vi.Committed == "" {
		vi.Committed = consts.FalseString
	}
	if vi.Items == nil {
		vi.Items = []RecognizeItem{}
	}
	errRunes := []rune(vi.ErrorMessage)
	if len(errRunes) > consts.RecognizeErrorMaxLen {
		vi.ErrorMessage = string(errRunes[:consts.RecognizeErrorMaxLen])
	}
	vi.SumCent = 0
	for i := range vi.Items {
		vi.SumCent += vi.Items[i].AmountCent
	}
	b, _ := json.Marshal(vi.Items)
	vi.ItemsJSON = string(b)
	return vi
}

// Deserialize 还原文本列里的识别项。
func (vi *RecognizeTask) Deserialize() *RecognizeTask {
	if vi == nil {
		return &RecognizeTask{Items: []RecognizeItem{}}
	}
	vi.Items = []RecognizeItem{}
	if vi.ItemsJSON != "" {
		// 文本列损坏时按空结果展示，任务状态与错误信息仍可读。
		_ = json.Unmarshal([]byte(vi.ItemsJSON), &vi.Items)
	}
	if vi.Committed == "" {
		vi.Committed = consts.FalseString
	}
	vi.CreatedAt = utils.MilliTimestamp(vi.CreatedAt)
	vi.UpdatedAt = utils.MilliTimestamp(vi.UpdatedAt)
	vi.FinishedAt = utils.MilliTimestamp(vi.FinishedAt)
	return vi
}

func (vi *RecognizeTask) Check() error {
	if vi == nil {
		return fmt.Errorf("识别任务为空")
	}
	if vi.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if vi.ImagePath == "" {
		return fmt.Errorf("imagePath is empty")
	}
	switch vi.Status {
	case consts.TaskStatusPending, consts.TaskStatusRunning, consts.TaskStatusSucceeded, consts.TaskStatusFailed:
	default:
		return fmt.Errorf("unsupported task status %q", vi.Status)
	}
	validCommitted := vi.Committed == consts.TrueString || vi.Committed == consts.FalseString
	if !validCommitted {
		return fmt.Errorf("unsupported committed %q", vi.Committed)
	}
	return nil
}

// ToUpdater 供识别结束时整体写回：items 文本列与 sum_cent 由 Serialize 派生。
func (vi *RecognizeTask) ToUpdater() map[string]any {
	return map[string]any{
		"status":        vi.Status,
		"model":         vi.Model,
		"raw_response":  vi.RawResponse,
		"items":         vi.ItemsJSON,
		"sum_cent":      vi.SumCent,
		"error_message": vi.ErrorMessage,
		"finished_at":   vi.FinishedAt,
		"updated_at":    time.Now().UTC().UnixMilli(),
	}
}

// CreateRecognizeTaskParam 携带上传的图片；ImageMIME 与 ImageSHA256 由 Serialize 派生。
type CreateRecognizeTaskParam struct {
	OwnerID     int64  `json:"-"`
	ImageData   []byte `json:"-"`
	ImageMIME   string `json:"-"`
	ImageSHA256 string `json:"-"`
}

func (p *CreateRecognizeTaskParam) Serialize() *CreateRecognizeTaskParam {
	if p == nil {
		return &CreateRecognizeTaskParam{ImageData: []byte{}}
	}
	if p.ImageData == nil {
		p.ImageData = []byte{}
	}
	p.ImageMIME = http.DetectContentType(p.ImageData)
	sum := sha256.Sum256(p.ImageData)
	p.ImageSHA256 = hex.EncodeToString(sum[:])
	return p
}

func (p *CreateRecognizeTaskParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if len(p.ImageData) == 0 {
		return fmt.Errorf("请选择一张礼簿照片")
	}
	if int64(len(p.ImageData)) > consts.RecognizeImageMaxSize {
		return fmt.Errorf("图片不能超过 5MB")
	}
	if _, ok := consts.RecognizeImageExt[p.ImageMIME]; !ok {
		return fmt.Errorf("只支持 jpg、png 图片")
	}
	return nil
}

// GetRecognizeTaskParam 用于任务详情与原图读取。
type GetRecognizeTaskParam struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"-"`
}

func (p *GetRecognizeTaskParam) Serialize() *GetRecognizeTaskParam {
	if p == nil {
		return &GetRecognizeTaskParam{}
	}
	return p
}

func (p *GetRecognizeTaskParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少识别任务 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	return nil
}

// CommitRecognizeItem 是核对页提交的一行；PersonID 为正时直接使用，否则按姓名与备注匹配或新建。
type CommitRecognizeItem struct {
	PersonID     int64  `json:"personId"`
	PersonName   string `json:"personName"`
	PersonRemark string `json:"personRemark"`
	AmountCent   int64  `json:"amountCent"`
	Note         string `json:"note"`
}

// CommitRecognizeTaskParam：礼簿、方向、事情与日期对整批生效。
type CommitRecognizeTaskParam struct {
	ID         int64                 `json:"id"`
	OwnerID    int64                 `json:"-"`
	BookID     int64                 `json:"bookId"`
	Direction  string                `json:"direction"`
	Matter     string                `json:"matter"`
	OccurredOn string                `json:"occurredOn"`
	Items      []CommitRecognizeItem `json:"items"`
}

func (p *CommitRecognizeTaskParam) Serialize() *CommitRecognizeTaskParam {
	if p == nil {
		return &CommitRecognizeTaskParam{Items: []CommitRecognizeItem{}}
	}
	p.Direction = strings.TrimSpace(p.Direction)
	p.Matter = strings.TrimSpace(p.Matter)
	p.OccurredOn = strings.TrimSpace(p.OccurredOn)
	if p.Direction == "" {
		p.Direction = consts.DirectionIn
	}
	if p.OccurredOn == "" {
		p.OccurredOn = utils.DateString(time.Now())
	}
	if p.Direction == consts.DirectionOut {
		p.BookID = 0
	}
	if p.Items == nil {
		p.Items = []CommitRecognizeItem{}
	}
	for i := range p.Items {
		p.Items[i].PersonName = strings.TrimSpace(p.Items[i].PersonName)
		p.Items[i].PersonRemark = strings.TrimSpace(p.Items[i].PersonRemark)
		p.Items[i].Note = strings.TrimSpace(p.Items[i].Note)
	}
	return p
}

func (p *CommitRecognizeTaskParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.ID <= 0 {
		return fmt.Errorf("缺少识别任务 id")
	}
	if p.OwnerID <= 0 {
		return fmt.Errorf("ownerID must be positive")
	}
	if !consts.ValidDirection(p.Direction) {
		return fmt.Errorf("方向只能是收或送")
	}
	inWithoutBook := p.Direction == consts.DirectionIn && p.BookID <= 0
	if inWithoutBook {
		return fmt.Errorf("收礼需要选择礼簿")
	}
	if _, err := utils.ParseDate(p.OccurredOn); err != nil {
		return fmt.Errorf("日期格式应为 YYYY-MM-DD")
	}
	if len(p.Items) == 0 {
		return fmt.Errorf("请至少勾选一条识别结果")
	}
	for i := range p.Items {
		missingPerson := p.Items[i].PersonID <= 0 && p.Items[i].PersonName == ""
		if missingPerson {
			return fmt.Errorf("第 %d 行缺少姓名", i+1)
		}
		if p.Items[i].AmountCent < 0 {
			return fmt.Errorf("第 %d 行金额不能为负数", i+1)
		}
	}
	return nil
}

// SearchRecognizeTaskParam：Statuses 为空表示不限状态，CreatedFrom 为毫秒下界（含）。
type SearchRecognizeTaskParam struct {
	OwnerID     int64               `json:"-"`
	TaskIDs     []int64             `json:"taskIds"`
	Statuses    []string            `json:"statuses"`
	CreatedFrom int64               `json:"createdFrom"`
	Filter      *modelfilter.Filter `json:"filter"`
}

func (p *SearchRecognizeTaskParam) Serialize() *SearchRecognizeTaskParam {
	if p == nil {
		return &SearchRecognizeTaskParam{Filter: &modelfilter.Filter{}}
	}
	if p.TaskIDs == nil {
		p.TaskIDs = []int64{}
	}
	if p.Statuses == nil {
		p.Statuses = []string{}
	}
	if p.Filter == nil {
		p.Filter = &modelfilter.Filter{}
	}
	noSort := p.Filter.SortField == "" && len(p.Filter.OrderByColumns) == 0
	if noSort {
		p.Filter.OrderByColumns = []clause.OrderByColumn{
			{Column: clause.Column{Name: "id"}, Desc: true},
		}
	}
	return p
}

// Check 允许 OwnerID 为 0：服务启动时清理残留任务不区分 owner。
func (p *SearchRecognizeTaskParam) Check() error {
	if p == nil {
		return fmt.Errorf("请求参数为空")
	}
	if p.OwnerID < 0 {
		return fmt.Errorf("ownerID must not be negative")
	}
	if p.Filter.SortField != "" {
		return fmt.Errorf("识别任务不支持自定义排序")
	}
	return nil
}

// CommitRecognizeResult 是批量写入的结果，金额单位为分。
type CommitRecognizeResult struct {
	RecordIDs  []int64 `json:"recordIds"`
	Count      int64   `json:"count"`
	NewPersons int64   `json:"newPersons"`
	SumCent    int64   `json:"sumCent"`
}
