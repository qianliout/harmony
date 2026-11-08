package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"harmony/component/config"
	"harmony/component/recognize/store"
	recordservice "harmony/component/record/service"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/person"
	"harmony/infra/model/recognize"
	"harmony/infra/model/record"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

const maxImageSize = 5 << 20

type RecordWriter interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, param record.CreateParam) (*record.Record, error)
	RepayInTx(ctx context.Context, tx *gorm.DB, param record.RepayParam) ([]record.Record, error)
}

type PersonStore interface {
	FindExactByName(db *gorm.DB, ownerID uint64, name string) ([]person.Person, error)
	CountAll(db *gorm.DB, ownerID uint64) (int64, error)
}

type PageCheckStore interface {
	UpsertPageCheck(db *gorm.DB, ownerID uint64, bookID uint64, page string, declaredCent int64) error
}

type RecognizeSrv struct {
	rdb      *rdb.RDB
	store    *store.Store
	records  RecordWriter
	persons  PersonStore
	pages    PageCheckStore
	provider Recognizer
	cfg      config.RecognizeConfig
	uploadDir string
	sem      chan struct{}
	log      *logger.Logger
}

func NewRecognizeSrv(
	database *rdb.RDB,
	taskStore *store.Store,
	recordSrv *recordservice.RecordSrv,
	personStore PersonStore,
	pageStore PageCheckStore,
	provider Recognizer,
	cfg config.RecognizeConfig,
	uploadDir string,
) *RecognizeSrv {
	return &RecognizeSrv{
		rdb:       database,
		store:     taskStore,
		records:   recordSrv,
		persons:   personStore,
		pages:     pageStore,
		provider:  provider,
		cfg:       cfg,
		uploadDir: uploadDir,
		sem:       make(chan struct{}, 2),
		log:       logger.New(logger.WithModule("recognize"), logger.WithSubModule("service")),
	}
}

type taskPayload struct {
	Page          string            `json:"page"`
	PageTotal     *float64          `json:"pageTotal"`
	PageTotalCent int64             `json:"pageTotalCent"`
	SumCent       int64             `json:"sumCent"`
	Items         []recognize.Item  `json:"items"`
}

func (s *RecognizeSrv) CreateTask(ctx context.Context, file io.Reader) (*recognize.Resp, error) {
	if s.cfg.APIKey == "" {
		return nil, response.BadGateway("识别服务还没配置 API Key，先在环境变量里配置")
	}
	ownerID := middleware.OwnerID(ctx)
	data, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil {
		return nil, response.BadRequest("读取图片失败", "read image failed")
	}
	if len(data) == 0 {
		return nil, response.BadRequest("请选择一张礼簿照片", "image is empty")
	}
	if len(data) > maxImageSize {
		return nil, response.BadRequest("图片不能超过 5MB", "image is too large")
	}
	mime := http.DetectContentType(data)
	ext := ""
	switch mime {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	default:
		return nil, response.BadRequest("只支持 jpg、png 图片", "unsupported image type")
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	month := time.Now().Format("200601")
	relPath := filepath.Join(month, sha+"."+ext)
	fullPath := filepath.Join(s.uploadDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, err
	}

	task := &recognize.Task{
		OwnerID:     ownerID,
		ImagePath:   relPath,
		ImageSHA256: sha,
		ImageSize:   len(data),
		Status:      consts.TaskStatusPending,
		Provider:    s.cfg.Provider,
		Model:       s.cfg.Model,
		// JSON 列不接受空字符串；pending 任务先写一个空对象。
		Items:       "{}",
	}
	if err := s.store.Create(s.rdb.Get(), task); err != nil {
		return nil, err
	}
	imageData := append([]byte(nil), data...)
	go s.runTask(task.ID, ownerID, Image{Data: imageData, MIME: mime})
	s.log.Info(ctx).Uint64("taskId", task.ID).Int("size", len(data)).Msg("recognize.create")
	return buildResp(task, taskPayload{Items: []recognize.Item{}}), nil
}

func (s *RecognizeSrv) runTask(taskID uint64, ownerID uint64, img Image) {
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	timeout := time.Duration(s.cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	db := s.rdb.Get()
	if err := s.store.MarkRunning(db, taskID); err != nil {
		s.log.Error(ctx).Err(err).Uint64("taskId", taskID).Msg("mark running failed")
		return
	}
	start := time.Now()
	result, err := s.provider.Recognize(ctx, img)
	if err != nil {
		message := err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			message = "识别超时"
		}
		if len(message) > 500 {
			message = message[:500]
		}
		if markErr := s.store.MarkFailed(db, taskID, message, ""); markErr != nil {
			s.log.Error(ctx).Err(markErr).Uint64("taskId", taskID).Msg("mark failed error")
		}
		s.log.Warn(ctx).Uint64("taskId", taskID).Err(err).Msg("recognize failed")
		return
	}
	payload := s.enhance(ownerID, result)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		_ = s.store.MarkFailed(db, taskID, "识别结果保存失败", result.Raw)
		return
	}
	if err := s.store.MarkSucceeded(db, taskID, string(payloadJSON), result.Raw); err != nil {
		s.log.Error(ctx).Err(err).Uint64("taskId", taskID).Msg("mark succeeded failed")
		return
	}
	s.log.Info(ctx).
		Uint64("taskId", taskID).
		Str("model", result.Model).
		Int64("latencyMs", time.Since(start).Milliseconds()).
		Int("itemCount", len(payload.Items)).
		Msg("recognize.succeeded")
}

func (s *RecognizeSrv) enhance(ownerID uint64, result *Result) taskPayload {
	payload := taskPayload{
		Page:      result.Page,
		PageTotal: result.PageTotal,
		Items:     make([]recognize.Item, 0, len(result.Items)),
	}
	if result.PageTotal != nil {
		payload.PageTotalCent = int64(math.Round(*result.PageTotal * 100))
	}
	for i := range result.Items {
		providerItem := result.Items[i]
		item := recognize.Item{
			Seq:         providerItem.Seq,
			Name:        strings.TrimSpace(providerItem.Name),
			AmountUpper: strings.TrimSpace(providerItem.AmountUpper),
			AmountLower: providerItem.AmountLower,
			GiftName:    strings.TrimSpace(providerItem.GiftName),
			Date:        strings.TrimSpace(providerItem.Date),
			Note:        strings.TrimSpace(providerItem.Note),
			Raw:         strings.TrimSpace(providerItem.Raw),
			Confidence:  clampConfidence(providerItem.Confidence),
		}
		if item.Seq == 0 {
			item.Seq = i + 1
		}
		if providerItem.ItemKind == consts.ItemKindCash || providerItem.ItemKind == consts.ItemKindHuopao || providerItem.ItemKind == consts.ItemKindGift {
			item.ItemKind = providerItem.ItemKind
		}
		// “贺礼/礼金”这类字样不是礼品；只有名目为实物礼品时才保留礼品名。
		if item.ItemKind == "" && item.GiftName != "" && providerItem.Amount == 0 {
			item.ItemKind = consts.ItemKindGift
		}
		if item.ItemKind != consts.ItemKindGift {
			item.GiftName = ""
		}
		if consts.ValidDirection(providerItem.Direction) {
			item.Direction = providerItem.Direction
		}
		amount := providerItem.Amount
		if amount < 0 {
			amount = 0
		}
		if upperYuan, ok := parseUpperAmount(item.AmountUpper); ok {
			if item.AmountLower != nil && math.Abs(upperYuan-*item.AmountLower) > 0.005 {
				item.AmountMismatch = true
				item.Confidence = math.Min(item.Confidence, 0.5)
			}
			amount = upperYuan
		} else if item.AmountLower != nil {
			amount = *item.AmountLower
		}
		item.Amount = amount
		if item.Note != "" {
			parsed := recordservice.ParsePaperNoteText(item.Note)
			item.RepayDrafts = parsed.Repays
		}
		if item.Name != "" {
			matches, err := s.persons.FindExactByName(s.rdb.Get(), ownerID, item.Name)
			if err == nil {
				match := &recognize.Match{Status: "none", Candidates: []recognize.PersonMatch{}}
				switch len(matches) {
				case 1:
					match.Status = "single"
					match.PersonID = matches[0].ID
				case 0:
				default:
					match.Status = "multiple"
				}
				for j := range matches {
					match.Candidates = append(match.Candidates, recognize.PersonMatch{
						ID:            matches[j].ID,
						Name:          matches[j].Name,
						RelationLabel: person.LabelOf(matches[j].Relation),
					})
				}
				item.Match = match
			}
		}
		payload.Items = append(payload.Items, item)
		payload.SumCent += int64(math.Round(item.Amount * 100))
	}
	return payload
}

func parseUpperAmount(upper string) (float64, bool) {
	text := strings.TrimSpace(upper)
	if text == "" {
		return 0, false
	}
	text = strings.NewReplacer("元", "", "圆", "", "整", "", " ", "").Replace(text)
	if text == "" {
		return 0, false
	}
	if number, ok := utils.ParseChineseNumber(text); ok {
		return float64(number), true
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func (s *RecognizeSrv) Detail(ctx context.Context, id uint64) (*recognize.Resp, error) {
	ownerID := middleware.OwnerID(ctx)
	task, err := s.store.Get(s.rdb.Get(), ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("识别任务不存在")
		}
		return nil, err
	}
	return buildResp(task, parsePayload(task.Items)), nil
}

func (s *RecognizeSrv) List(ctx context.Context, page, pageSize int64) ([]recognize.Resp, int64, error) {
	ownerID := middleware.OwnerID(ctx)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > consts.MaxPageSize {
		pageSize = consts.DefaultPageSize
	}
	tasks, total, err := s.store.List(s.rdb.Get(), ownerID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]recognize.Resp, 0, len(tasks))
	for i := range tasks {
		out = append(out, *buildResp(&tasks[i], parsePayload(tasks[i].Items)))
	}
	return out, total, nil
}

func (s *RecognizeSrv) Commit(ctx context.Context, id uint64, param recognize.CommitParam) (*recognize.CommitResult, error) {
	ownerID := middleware.OwnerID(ctx)
	db := s.rdb.Get()
	task, err := s.store.Get(db, ownerID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFound("识别任务不存在")
		}
		return nil, err
	}
	if task.Status != consts.TaskStatusSucceeded {
		return nil, response.StateConflict("识别还没有完成，不能写入")
	}
	if task.Committed {
		return nil, response.StateConflict("这批识别结果已经写入过了")
	}
	if len(param.Items) == 0 {
		return nil, response.BadRequest("请至少勾选一条识别结果", "no items selected")
	}
	beforePersons, err := s.persons.CountAll(db, ownerID)
	if err != nil {
		return nil, err
	}

	result := &recognize.CommitResult{RecordIDs: []uint64{}}
	err = db.Transaction(func(tx *gorm.DB) error {
		for index := range param.Items {
			item := param.Items[index]
			occurredOn := strings.TrimSpace(item.OccurredOn)
			if occurredOn == "" {
				occurredOn = utils.DateString(time.Now())
			}
			bookID := item.BookID
			if bookID == 0 {
				bookID = param.BookID
			}
			if bookID == 0 {
				return commitItemError(index, "bookId", "还没有选择礼簿")
			}
			paperPage := item.PaperPage
			paperSeq := item.PaperSeq
			if paperPage == "" && parsePayload(task.Items).Page != "" {
				paperPage = parsePayload(task.Items).Page
			}
			created, err := s.records.CreateInTx(ctx, tx, record.CreateParam{
				PersonID:        item.PersonID,
				PersonName:      item.PersonName,
				EventID:         item.EventID,
				BookID:          bookID,
				ItemKind:        item.ItemKind,
				Direction:       defaultDirection(item.Direction),
				AmountCent:      item.AmountCent,
				GiftName:        item.GiftName,
				GiftValueCent:   item.GiftValueCent,
				OccurredOn:      occurredOn,
				DatePrecision:   defaultPrecision(item.DatePrecision),
				Status:          item.Status,
				Remark:          item.Remark,
				PaperNote:       item.PaperNote,
				PaperPage:       paperPage,
				PaperSeq:        paperSeq,
				Source:          consts.RecordSourceRecognize,
				RecognizeTaskID: &id,
				Confirm:         true,
			})
			if err != nil {
				return wrapItemError(index, err)
			}
			result.RecordIDs = append(result.RecordIDs, created.ID)
			result.Count++
			result.SumCent += created.TotalCent
			for draftIndex := range item.Repays {
				draft := item.Repays[draftIndex]
				repayParam := record.RepayParam{
					OriginIDs:     []uint64{created.ID},
					Event:         draft.Event,
					OccurredOn:    draft.OccurredOn,
					DatePrecision: draft.DatePrecision,
					PaperNote:     item.PaperNote,
					Items:         draft.Items,
					Remark:        "",
				}
				repayRecords, err := s.records.RepayInTx(ctx, tx, repayParam)
				if err != nil {
					return wrapItemError(index, err)
				}
				result.RepayCount += len(repayRecords)
			}
		}
		pageTotalCent := int64(0)
		hasPageTotal := false
		if param.PageTotalCent != nil {
			pageTotalCent = *param.PageTotalCent
			hasPageTotal = true
		} else if payload := parsePayload(task.Items); payload.PageTotalCent > 0 {
			pageTotalCent = payload.PageTotalCent
			hasPageTotal = true
		}
		page := param.Page
		if page == "" {
			page = parsePayload(task.Items).Page
		}
		if hasPageTotal && page != "" {
			bookID := param.BookID
			if bookID == 0 {
				bookID = param.Items[0].BookID
			}
			if bookID > 0 {
				if err := s.pages.UpsertPageCheck(tx, ownerID, bookID, page, pageTotalCent); err != nil {
					return err
				}
			}
		}
		return s.store.MarkCommitted(tx, id)
	})
	if err != nil {
		return nil, err
	}
	afterPersons, err := s.persons.CountAll(db, ownerID)
	if err == nil {
		result.NewPersons = int(afterPersons - beforePersons)
	}
	s.log.Info(ctx).Uint64("taskId", id).Int("count", result.Count).Msg("recognize.commit")
	return result, nil
}

func (s *RecognizeSrv) FailStale(ctx context.Context) error {
	return s.store.FailStale(s.rdb.Get())
}

func buildResp(task *recognize.Task, payload taskPayload) *recognize.Resp {
	return &recognize.Resp{
		ID:            task.ID,
		Status:        task.Status,
		Provider:      task.Provider,
		Model:         task.Model,
		ImagePath:     task.ImagePath,
		ImageURL:      "/uploads/" + strings.ReplaceAll(task.ImagePath, string(os.PathSeparator), "/"),
		ImageSHA256:   task.ImageSHA256,
		ImageSize:     task.ImageSize,
		ErrorMessage:  task.ErrorMessage,
		Committed:     task.Committed,
		Page:          payload.Page,
		PageTotalCent: payload.PageTotalCent,
		SumCent:       payload.SumCent,
		Items:         payload.Items,
		CreatedAt:     utils.Millis(task.CreatedAt),
		FinishedAt:    utils.MillisPtr(task.FinishedAt),
	}
}

func parsePayload(raw string) taskPayload {
	payload := taskPayload{Items: []recognize.Item{}}
	if strings.TrimSpace(raw) == "" {
		return payload
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return taskPayload{Items: []recognize.Item{}}
	}
	if payload.Items == nil {
		payload.Items = []recognize.Item{}
	}
	return payload
}

func commitItemError(index int, field, message string) *response.ErrI18 {
	err := response.BadRequest(message, message)
	err.Custom = map[string]any{
		"itemErrors": []map[string]any{{
			"index": index, "field": field, "message": message,
		}},
	}
	return err
}

func wrapItemError(index int, err error) error {
	var errI18 *response.ErrI18
	if errors.As(err, &errI18) {
		message := errI18.Msg
		if message == "" {
			message = errI18.Error()
		}
		return commitItemError(index, "item", message)
	}
	return fmt.Errorf("第 %d 行写入失败: %w", index+1, err)
}

func defaultDirection(value string) string {
	if consts.ValidDirection(value) {
		return value
	}
	return consts.DirectionIn
}

func defaultPrecision(value string) string {
	if consts.ValidDatePrecision(value) {
		return value
	}
	return consts.DatePrecisionDay
}
