package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	bookstore "harmony/component/book/store"
	"harmony/component/config"
	personservice "harmony/component/person/service"
	personstore "harmony/component/person/store"
	"harmony/component/recognize/store"
	recordservice "harmony/component/record/service"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/model/book"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/model/recognize"
	"harmony/infra/model/record"
	"harmony/infra/response"
)

// recognizer 是 RecognizeSrv 对识别模型的最小依赖，由 *OpenAICompatRecognizer 满足，测试可替换。
type recognizer interface {
	Recognize(ctx context.Context, img Image) (*Result, error)
}

// commitPerson 是写入某一行时确定的亲友；Created 表示本次批量写入新建了该亲友。
type commitPerson struct {
	personID int64
	created  bool
}

type RecognizeSrv struct {
	taskDal    *store.RecognizeTaskDao
	personDal  *personstore.PersonDao
	bookDal    *bookstore.BookDao
	personSrv  *personservice.PersonSrv
	recordSrv  *recordservice.RecordSrv
	recognizer recognizer
	cfg        config.RecognizeConfig
	uploadDir  string
	sem        chan struct{}
	log        *logger.Logger
}

func NewRecognizeSrv(
	taskDal *store.RecognizeTaskDao,
	personDal *personstore.PersonDao,
	bookDal *bookstore.BookDao,
	personSrv *personservice.PersonSrv,
	recordSrv *recordservice.RecordSrv,
	recognizer recognizer,
	cfg config.RecognizeConfig,
	uploadDir string,
) *RecognizeSrv {
	return &RecognizeSrv{
		taskDal:    taskDal,
		personDal:  personDal,
		bookDal:    bookDal,
		personSrv:  personSrv,
		recordSrv:  recordSrv,
		recognizer: recognizer,
		cfg:        cfg,
		uploadDir:  uploadDir,
		sem:        make(chan struct{}, consts.RecognizeConcurrency),
		log:        logger.New(logger.WithModule("recognize"), logger.WithSubModule("service")),
	}
}

// CreateRecognizeTask 保存原图、建任务，并在后台调用识别模型；接口立即返回 pending 任务。
func (s *RecognizeSrv) CreateRecognizeTask(ctx context.Context, param *recognize.CreateRecognizeTaskParam) (*recognize.RecognizeTask, error) {
	if s.cfg.APIKey == "" {
		return nil, response.BadGateway("识别服务还没配置 API Key，先在环境变量里配置", "recognize api key is not configured")
	}
	if err := s.checkDailyLimit(ctx, param.OwnerID); err != nil {
		return nil, err
	}

	ext := consts.RecognizeImageExt[param.ImageMIME]
	relPath := filepath.Join(time.Now().Format("200601"), param.ImageSHA256+"."+ext)
	fullPath := filepath.Join(s.uploadDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		s.log.Error(ctx).Err(err).Str("path", fullPath).Msg("create upload dir failed")
		return nil, err
	}
	if err := os.WriteFile(fullPath, param.ImageData, 0o644); err != nil {
		s.log.Error(ctx).Err(err).Str("path", fullPath).Msg("write upload image failed")
		return nil, err
	}

	task := &recognize.RecognizeTask{
		OwnerID:     param.OwnerID,
		ImagePath:   relPath,
		ImageSHA256: param.ImageSHA256,
		ImageSize:   int64(len(param.ImageData)),
		Status:      consts.TaskStatusPending,
		Provider:    s.cfg.Provider,
		Model:       s.cfg.Model,
	}
	if err := s.taskDal.CreateRecognizeTask(ctx, task); err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Msg("createRecognizeTask failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("taskID", task.ID).Int64("size", task.ImageSize).Msg("recognize task created")

	background := *task
	img := Image{Data: param.ImageData, MIME: param.ImageMIME}
	go s.runRecognizeTask(&background, img)
	return task, nil
}

func (s *RecognizeSrv) GetRecognizeTask(ctx context.Context, param *recognize.GetRecognizeTaskParam) (*recognize.RecognizeTask, error) {
	task, err := s.getTask(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// ResolveRecognizeImage 返回原图的绝对路径，并保证它落在上传目录内。
func (s *RecognizeSrv) ResolveRecognizeImage(ctx context.Context, param *recognize.GetRecognizeTaskParam) (string, error) {
	task, err := s.getTask(ctx, param.OwnerID, param.ID)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(s.uploadDir)
	if err != nil {
		s.log.Error(ctx).Err(err).Str("uploadDir", s.uploadDir).Msg("resolve upload dir failed")
		return "", err
	}
	target := filepath.Clean(filepath.Join(root, task.ImagePath))
	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return "", response.Forbidden("原图路径无效", "invalid image path")
	}
	return target, nil
}

// CommitRecognizeTask 把核对后的识别结果写成记录。不使用事务：先完成全部只读校验，再逐条写入；
// 中途写入失败时已写入的记录保留，任务不标记为已写入，错误信息带上失败行号。
func (s *RecognizeSrv) CommitRecognizeTask(ctx context.Context, param *recognize.CommitRecognizeTaskParam) (*recognize.CommitRecognizeResult, error) {
	task, err := s.getTask(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	if task.Status != consts.TaskStatusSucceeded {
		return nil, response.StateConflict("识别还没有完成，不能写入", "task is not succeeded")
	}
	if task.Committed == consts.TrueString {
		return nil, response.StateConflict("这批识别结果已经写入过了", "task already committed")
	}
	if param.Direction == consts.DirectionIn {
		if err := s.checkBook(ctx, param.OwnerID, param.BookID); err != nil {
			return nil, err
		}
	}
	matched, err := s.matchCommitPersons(ctx, param)
	if err != nil {
		return nil, err
	}

	result := &recognize.CommitRecognizeResult{RecordIDs: make([]int64, 0, len(param.Items))}
	createdPersons := make(map[string]int64)
	for i, item := range param.Items {
		resolved, err := s.ensureCommitPerson(ctx, param.OwnerID, item, matched[i], createdPersons)
		if err != nil {
			s.log.Error(ctx).Err(err).Int64("taskID", param.ID).Ints64("recordIDs", result.RecordIDs).
				Msg("commit recognize task stopped")
			return nil, s.lineError(i, err)
		}
		if resolved.created {
			result.NewPersons++
		}
		createParam := &record.CreateRecordParam{
			OwnerID:    param.OwnerID,
			PersonID:   resolved.personID,
			BookID:     param.BookID,
			Direction:  param.Direction,
			AmountCent: item.AmountCent,
			Matter:     param.Matter,
			Note:       item.Note,
			OccurredOn: param.OccurredOn,
		}
		view, err := s.recordSrv.CreateRecord(ctx, createParam)
		if err != nil {
			s.log.Error(ctx).Err(err).Int64("taskID", param.ID).Ints64("recordIDs", result.RecordIDs).
				Msg("commit recognize task stopped")
			return nil, s.lineError(i, err)
		}
		result.RecordIDs = append(result.RecordIDs, view.ID)
		result.Count++
		result.SumCent += view.AmountCent
	}

	updater := map[string]any{
		"committed":  consts.TrueString,
		"updated_at": time.Now().UTC().UnixMilli(),
	}
	if err := s.taskDal.UpdateRecognizeTask(ctx, param.ID, updater); err != nil {
		s.log.Error(ctx).Err(err).Int64("taskID", param.ID).Msg("mark recognize task committed failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("taskID", param.ID).Int64("count", result.Count).Msg("recognize task committed")
	return result, nil
}

// FailStaleRecognizeTask 在服务启动时把上次未完成的任务置为失败，提示用户重新上传。
func (s *RecognizeSrv) FailStaleRecognizeTask(ctx context.Context) error {
	searchParam := &recognize.SearchRecognizeTaskParam{
		Statuses: []string{consts.TaskStatusPending, consts.TaskStatusRunning},
	}
	tasks, _, err := s.taskDal.SearchRecognizeTask(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Msg("search stale recognize tasks failed")
		return err
	}
	now := time.Now().UTC().UnixMilli()
	for _, task := range tasks {
		updater := map[string]any{
			"status":        consts.TaskStatusFailed,
			"error_message": "服务重启，识别未完成，请重新上传",
			"finished_at":   now,
			"updated_at":    now,
		}
		if err := s.taskDal.UpdateRecognizeTask(ctx, task.ID, updater); err != nil {
			s.log.Error(ctx).Err(err).Int64("taskID", task.ID).Msg("fail stale recognize task failed")
			return err
		}
	}
	return nil
}

// runRecognizeTask 在后台 goroutine 执行识别。请求 ctx 在响应后即结束，因此以 Background 为根，
// 用配置的超时约束总时长；并发数由 sem 限制。
func (s *RecognizeSrv) runRecognizeTask(task *recognize.RecognizeTask, img Image) {
	timeout := time.Duration(s.cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			s.log.Error(ctx).Int64("taskID", task.ID).Interface("panic", r).Str("stack", string(debug.Stack())).
				Msg("recognize task panicked")
		}
	}()

	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	runningUpdater := map[string]any{
		"status":     consts.TaskStatusRunning,
		"updated_at": time.Now().UTC().UnixMilli(),
	}
	if err := s.taskDal.UpdateRecognizeTask(ctx, task.ID, runningUpdater); err != nil {
		s.log.Error(ctx).Err(err).Int64("taskID", task.ID).Msg("mark recognize task running failed")
		return
	}

	start := time.Now()
	result, err := s.recognizer.Recognize(ctx, img)
	task.FinishedAt = time.Now().UTC().UnixMilli()
	if err != nil {
		s.log.Warn(ctx).Err(err).Int64("taskID", task.ID).Msg("recognize failed")
		task.Status = consts.TaskStatusFailed
		task.ErrorMessage = err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			task.ErrorMessage = "识别超时，请重试"
		}
		s.saveRecognizeResult(ctx, task)
		return
	}

	items, err := s.matchRecognizeItems(ctx, task.OwnerID, result.Items)
	if err != nil {
		task.Status = consts.TaskStatusFailed
		task.ErrorMessage = "识别结果匹配亲友失败，请重试"
		s.saveRecognizeResult(ctx, task)
		return
	}
	task.Status = consts.TaskStatusSucceeded
	task.Model = result.Model
	task.RawResponse = result.Raw
	task.Items = items
	s.saveRecognizeResult(ctx, task)
	s.log.Info(ctx).Int64("taskID", task.ID).Str("model", result.Model).
		Int64("latencyMs", time.Since(start).Milliseconds()).Int("itemCount", len(items)).
		Msg("recognize succeeded")
}

// saveRecognizeResult 整体写回识别结果；items 文本列与合计由 Serialize 派生，因此走 Serialize → Check → ToUpdater。
func (s *RecognizeSrv) saveRecognizeResult(ctx context.Context, task *recognize.RecognizeTask) {
	task = task.Serialize()
	if err := task.Check(); err != nil {
		s.log.Error(ctx).Err(err).Int64("taskID", task.ID).Msg("check recognize result failed")
		return
	}
	if err := s.taskDal.UpdateRecognizeTask(ctx, task.ID, task.ToUpdater()); err != nil {
		s.log.Error(ctx).Err(err).Int64("taskID", task.ID).Msg("save recognize result failed")
	}
}

// matchRecognizeItems 把模型输出转成识别项，金额转分、置信度转百分比，并标出同名亲友。
func (s *RecognizeSrv) matchRecognizeItems(ctx context.Context, ownerID int64, providerItems []ProviderItem) ([]recognize.RecognizeItem, error) {
	items := make([]recognize.RecognizeItem, 0, len(providerItems))
	personsByName, err := s.searchPersonsByName(ctx, ownerID, providerItems)
	if err != nil {
		return items, err
	}

	for i, providerItem := range providerItems {
		name := strings.TrimSpace(providerItem.Name)
		item := recognize.RecognizeItem{
			Seq:        providerItem.Seq,
			Name:       name,
			AmountCent: int64(math.Round(providerItem.Amount * 100)),
			Note:       strings.TrimSpace(providerItem.Note),
			Raw:        strings.TrimSpace(providerItem.Raw),
			Confidence: int64(math.Round(math.Max(0, math.Min(1, providerItem.Confidence)) * 100)),
			Match:      recognize.RecognizeMatch{Status: consts.MatchStatusNone, Candidates: []recognize.RecognizeCandidate{}},
		}
		if item.Seq <= 0 {
			item.Seq = int64(i + 1)
		}
		if item.AmountCent < 0 {
			item.AmountCent = 0
		}
		candidates := personsByName[name]
		for _, row := range candidates {
			item.Match.Candidates = append(item.Match.Candidates, recognize.RecognizeCandidate{
				PersonID:     row.ID,
				PersonName:   row.Name,
				PersonRemark: row.Remark,
			})
		}
		switch len(candidates) {
		case 0:
		case 1:
			item.Match.Status = consts.MatchStatusSingle
			item.Match.PersonID = candidates[0].ID
		default:
			item.Match.Status = consts.MatchStatusMultiple
		}
		items = append(items, item)
	}
	return items, nil
}

// searchPersonsByName 一次查出识别姓名对应的已有亲友；没有可用姓名时不查询，避免 Names 为空退化成全表。
func (s *RecognizeSrv) searchPersonsByName(ctx context.Context, ownerID int64, providerItems []ProviderItem) (map[string][]*person.Person, error) {
	personsByName := make(map[string][]*person.Person)
	names := make([]string, 0, len(providerItems))
	for _, providerItem := range providerItems {
		name := strings.TrimSpace(providerItem.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return personsByName, nil
	}
	searchParam := &person.SearchPersonParam{OwnerID: ownerID, Names: names}
	persons, _, err := s.personDal.SearchPerson(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", ownerID).Msg("search recognize persons failed")
		return personsByName, err
	}
	for _, row := range persons {
		personsByName[row.Name] = append(personsByName[row.Name], row)
	}
	return personsByName, nil
}

// matchCommitPersons 只读匹配每一行的亲友；返回与 Items 等长的切片，nil 表示需要新建。
func (s *RecognizeSrv) matchCommitPersons(ctx context.Context, param *recognize.CommitRecognizeTaskParam) ([]*person.Person, error) {
	matched := make([]*person.Person, 0, len(param.Items))
	for i, item := range param.Items {
		matchParam := &person.MatchPersonParam{
			OwnerID:      param.OwnerID,
			PersonID:     item.PersonID,
			PersonName:   item.PersonName,
			PersonRemark: item.PersonRemark,
		}
		row, err := s.personSrv.MatchPerson(ctx, matchParam)
		if err != nil {
			return matched, s.lineError(i, err)
		}
		matched = append(matched, row)
	}
	return matched, nil
}

// ensureCommitPerson 返回该行要用的亲友 id；同一批次里重复出现的新姓名只建一次。
func (s *RecognizeSrv) ensureCommitPerson(
	ctx context.Context,
	ownerID int64,
	item recognize.CommitRecognizeItem,
	matched *person.Person,
	createdPersons map[string]int64,
) (*commitPerson, error) {
	if matched != nil {
		return &commitPerson{personID: matched.ID}, nil
	}
	key := item.PersonName + consts.PersonDisplaySep + item.PersonRemark
	if personID, ok := createdPersons[key]; ok {
		return &commitPerson{personID: personID}, nil
	}
	createParam := &person.CreatePersonParam{OwnerID: ownerID, Name: item.PersonName, Remark: item.PersonRemark}
	created, err := s.personSrv.CreatePerson(ctx, createParam)
	if err != nil {
		return nil, err
	}
	createdPersons[key] = created.ID
	return &commitPerson{personID: created.ID, created: true}, nil
}

func (s *RecognizeSrv) checkDailyLimit(ctx context.Context, ownerID int64) error {
	if s.cfg.DailyLimit <= 0 {
		return nil
	}
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	searchParam := &recognize.SearchRecognizeTaskParam{
		OwnerID:     ownerID,
		Statuses:    []string{consts.TaskStatusPending, consts.TaskStatusRunning, consts.TaskStatusSucceeded},
		CreatedFrom: dayStart.UTC().UnixMilli(),
		Filter:      &modelfilter.Filter{Limit: 1},
	}
	_, used, err := s.taskDal.SearchRecognizeTask(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", ownerID).Msg("count today recognize tasks failed")
		return err
	}
	if used >= s.cfg.DailyLimit {
		return response.BadRequest("今天的识别次数已用完", "daily recognition limit reached")
	}
	return nil
}

func (s *RecognizeSrv) checkBook(ctx context.Context, ownerID, bookID int64) error {
	searchParam := &book.SearchBookParam{OwnerID: ownerID, BookIDs: []int64{bookID}}
	rows, _, err := s.bookDal.SearchBook(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("bookID", bookID).Msg("search book by id failed")
		return err
	}
	if len(rows) == 0 {
		return response.BadRequest("礼簿不存在", "book not found")
	}
	return nil
}

func (s *RecognizeSrv) getTask(ctx context.Context, ownerID, taskID int64) (*recognize.RecognizeTask, error) {
	searchParam := &recognize.SearchRecognizeTaskParam{OwnerID: ownerID, TaskIDs: []int64{taskID}}
	rows, _, err := s.taskDal.SearchRecognizeTask(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("taskID", taskID).Msg("search recognize task failed")
		return nil, err
	}
	if len(rows) == 0 {
		return nil, response.NotFound("识别任务不存在", "recognize task not found")
	}
	return rows[0], nil
}

// lineError 给逐行处理的错误加上行号，保留原错误码与 custom。
func (s *RecognizeSrv) lineError(index int, err error) error {
	prefix := fmt.Sprintf("第 %d 行：", index+1)
	var errI18 *response.ErrI18
	if !errors.As(err, &errI18) {
		return fmt.Errorf("%s%w", prefix, err)
	}
	wrapped := *errI18
	wrapped.Msg = prefix + errI18.Msg
	wrapped.ZH = prefix + errI18.ZH
	wrapped.EN = fmt.Sprintf("line %d: %s", index+1, errI18.EN)
	return &wrapped
}
