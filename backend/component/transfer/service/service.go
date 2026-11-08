package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"gorm.io/gorm"

	bookstore "harmony/component/book/store"
	dictstore "harmony/component/dict/store"
	eventstore "harmony/component/event/store"
	personstore "harmony/component/person/store"
	recordservice "harmony/component/record/service"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/book"
	"harmony/infra/model/event"
	modelfilter "harmony/infra/model/filter"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/model/transfer"
	"harmony/infra/rdb"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type RecordService interface {
	List(ctx context.Context, q record.Query, f *modelfilter.Filter) ([]record.Resp, int64, error)
	CreateInTx(ctx context.Context, tx *gorm.DB, param record.CreateParam) (*record.Record, error)
}

type TransferSrv struct {
	rdb     *rdb.RDB
	records RecordService
	persons *personstore.Store
	events  *eventstore.Store
	books   *bookstore.Store
	dicts   *dictstore.Store
	log     *logger.Logger
}

func NewTransferSrv(
	database *rdb.RDB,
	recordSrv *recordservice.RecordSrv,
	personStore *personstore.Store,
	eventStore *eventstore.Store,
	bookStore *bookstore.Store,
	dictStore *dictstore.Store,
) *TransferSrv {
	return &TransferSrv{
		rdb:     database,
		records: recordSrv,
		persons: personStore,
		events:  eventStore,
		books:   bookStore,
		dicts:   dictStore,
		log:     logger.New(logger.WithModule("transfer"), logger.WithSubModule("service")),
	}
}

var csvHeader = []string{"日期", "姓名", "关系", "事项", "事项类型", "事项日期", "本子", "名目", "方向", "现金", "礼品", "礼品估价", "状态", "备注"}

func (s *TransferSrv) Export(ctx context.Context, format string, q record.Query) (string, string, []byte, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "csv"
	}
	items, _, err := s.records.List(ctx, q, nil)
	if err != nil {
		return "", "", nil, err
	}
	ownerID := middleware.OwnerID(ctx)
	eventIDs := make([]uint64, 0, len(items))
	for i := range items {
		eventIDs = append(eventIDs, items[i].EventID)
	}
	events, err := s.events.GetMany(s.rdb.Get(), ownerID, dedupe(eventIDs))
	if err != nil {
		return "", "", nil, err
	}
	eventDateByID := make(map[uint64]string, len(events))
	for i := range events {
		eventDateByID[events[i].ID] = utils.DateStringOf(events[i].EventDate)
	}
	rows := make([]transfer.Row, 0, len(items))
	for i := range items {
		item := items[i]
		rows = append(rows, transfer.Row{
			Date:       item.OccurredOn,
			PersonName: item.PersonName,
			Relation:   relationLabel(item.PersonRelation),
			EventName:  item.EventName,
			EventType:  eventTypeLabel(item.EventType),
			EventDate:  eventDateByID[item.EventID],
			BookName:   item.BookName,
			ItemKind:   item.ItemKindLabel,
			Direction:  directionLabel(item.Direction),
			Amount:     utils.YuanString(item.AmountCent),
			GiftName:   item.GiftName,
			GiftValue:  utils.YuanString(item.GiftValueCent),
			Status:     item.StatusLabel,
			Remark:     item.Remark,
		})
	}
	if format == "json" {
		payload, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return "", "", nil, err
		}
		return "harmony-export.json", "application/json; charset=utf-8", payload, nil
	}
	buf := &bytes.Buffer{}
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(buf)
	if err := writer.Write(csvHeader); err != nil {
		return "", "", nil, err
	}
	for i := range rows {
		row := rows[i]
		values := []string{row.Date, row.PersonName, row.Relation, row.EventName, row.EventType, row.EventDate,
			row.BookName, row.ItemKind, row.Direction, row.Amount, row.GiftName, row.GiftValue, row.Status, row.Remark}
		if err := writer.Write(values); err != nil {
			return "", "", nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", "", nil, err
	}
	return "harmony-export.csv", "text/csv; charset=utf-8", buf.Bytes(), nil
}

func (s *TransferSrv) ImportPreview(ctx context.Context, file io.Reader) (*transfer.Preview, error) {
	raw, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		return nil, response.BadRequest("读取文件失败", "read file failed")
	}
	text := strings.TrimPrefix(string(raw), "\xEF\xBB\xBF")
	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, response.BadRequest("CSV 格式不正确", "invalid csv")
	}
	if len(records) == 0 {
		return nil, response.BadRequest("文件是空的", "empty file")
	}
	header := records[0]
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[strings.TrimSpace(name)] = i
	}
	for _, required := range []string{"日期", "姓名", "事项", "本子", "方向"} {
		if _, ok := index[required]; !ok {
			return nil, response.BadRequest("缺少列："+required, "missing column "+required)
		}
	}
	ownerID := middleware.OwnerID(ctx)
	itemLabels, err := s.itemKindLabelMap(ownerID)
	if err != nil {
		return nil, err
	}
	preview := &transfer.Preview{Rows: make([]transfer.PreviewRow, 0)}
	for lineIndex := 1; lineIndex < len(records); lineIndex++ {
		line := records[lineIndex]
		row := transfer.Row{
			Date:       cell(line, index, "日期"),
			PersonName: cell(line, index, "姓名"),
			Relation:   cell(line, index, "关系"),
			EventName:  cell(line, index, "事项"),
			EventType:  cell(line, index, "事项类型"),
			EventDate:  cell(line, index, "事项日期"),
			BookName:   cell(line, index, "本子"),
			ItemKind:   cell(line, index, "名目"),
			Direction:  cell(line, index, "方向"),
			Amount:     cell(line, index, "现金"),
			GiftName:   cell(line, index, "礼品"),
			GiftValue:  cell(line, index, "礼品估价"),
			Status:     cell(line, index, "状态"),
			Remark:     cell(line, index, "备注"),
		}
		if isEmptyRow(row) {
			continue
		}
		errorsList := validateRow(row, itemLabels)
		preview.Rows = append(preview.Rows, transfer.PreviewRow{
			Line:   lineIndex + 1,
			OK:     len(errorsList) == 0,
			Errors: errorsList,
			Row:    row,
		})
	}
	preview.Total = len(preview.Rows)
	for i := range preview.Rows {
		if preview.Rows[i].OK {
			preview.OKCount++
		} else {
			preview.ErrorCount++
		}
	}
	return preview, nil
}

func (s *TransferSrv) Commit(ctx context.Context, param transfer.CommitParam) (*transfer.CommitResult, error) {
	if len(param.Rows) == 0 {
		return nil, response.BadRequest("没有可导入的行", "no rows")
	}
	ownerID := middleware.OwnerID(ctx)
	itemLabels, err := s.itemKindLabelMap(ownerID)
	if err != nil {
		return nil, err
	}
	result := &transfer.CommitResult{}
	err = s.rdb.Transaction(func(tx *gorm.DB) error {
		for index := range param.Rows {
			row := param.Rows[index]
			if errorsList := validateRow(row, itemLabels); len(errorsList) > 0 {
				return fmt.Errorf("第 %d 行：%s", index+1, strings.Join(errorsList, "；"))
			}
			personItem, created, err := s.ensurePerson(tx, ownerID, row.PersonName, row.Relation)
			if err != nil {
				return fmt.Errorf("第 %d 行：%w", index+1, err)
			}
			if created {
				result.NewPersons++
			}
			eventItem, eventCreated, err := s.ensureEvent(tx, ownerID, row)
			if err != nil {
				return fmt.Errorf("第 %d 行：%w", index+1, err)
			}
			if eventCreated {
				result.NewEvents++
			}
			bookItem, bookCreated, err := s.ensureBook(tx, ownerID, row.BookName)
			if err != nil {
				return fmt.Errorf("第 %d 行：%w", index+1, err)
			}
			if bookCreated {
				result.NewBooks++
			}
			amountCent := int64(0)
			if strings.TrimSpace(row.Amount) != "" {
				amountCent, err = utils.ParseYuanToCent(row.Amount)
				if err != nil {
					return fmt.Errorf("第 %d 行：现金金额不正确", index+1)
				}
			}
			giftValueCent := int64(0)
			if strings.TrimSpace(row.GiftValue) != "" {
				giftValueCent, err = utils.ParseYuanToCent(row.GiftValue)
				if err != nil {
					return fmt.Errorf("第 %d 行：礼品估价不正确", index+1)
				}
			}
			direction := directionCode(row.Direction)
			status := statusCode(row.Status, directionCode(row.Direction))
			precision := consts.DatePrecisionDay
			if _, err := s.records.CreateInTx(ctx, tx, record.CreateParam{
				PersonID:      personItem.ID,
				EventID:       eventItem.ID,
				BookID:        bookItem.ID,
				ItemKind:      itemKindCode(row.ItemKind, itemLabels),
				Direction:     direction,
				AmountCent:    amountCent,
				GiftName:      strings.TrimSpace(row.GiftName),
				GiftValueCent: giftValueCent,
				OccurredOn:    strings.TrimSpace(row.Date),
				DatePrecision: precision,
				Status:        status,
				Remark:        strings.TrimSpace(row.Remark),
				Source:        consts.RecordSourceImport,
				Confirm:       true,
			}); err != nil {
				return fmt.Errorf("第 %d 行：%w", index+1, err)
			}
			result.Count++
		}
		return nil
	})
	if err != nil {
		return nil, response.BadRequest(err.Error(), err.Error())
	}
	s.log.Info(ctx).Int("count", result.Count).Msg("transfer.import")
	return result, nil
}

func (s *TransferSrv) ensurePerson(tx *gorm.DB, ownerID uint64, name string, relationText string) (*person.Person, bool, error) {
	name = strings.TrimSpace(name)
	matches, err := s.persons.FindExactByName(tx, ownerID, name)
	if err != nil {
		return nil, false, err
	}
	if len(matches) > 1 {
		return nil, false, fmt.Errorf("有 %d 个同名亲友 %s，先合并或改名再导入", len(matches), name)
	}
	if len(matches) == 1 {
		return &matches[0], false, nil
	}
	relation := relationCode(relationText)
	full, initials := utils.Pinyin(name)
	created := &person.Person{
		OwnerID:      ownerID,
		Name:         name,
		NamePinyin:   full,
		NameInitials: initials,
		Relation:     relation,
	}
	if err := s.persons.Create(tx, created); err != nil {
		return nil, false, err
	}
	return created, true, nil
}

func (s *TransferSrv) ensureEvent(tx *gorm.DB, ownerID uint64, row transfer.Row) (*event.Event, bool, error) {
	name := strings.TrimSpace(row.EventName)
	existing, err := s.events.FindByName(tx, ownerID, name)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	eventType := eventTypeCode(row.EventType)
	if eventType == "" || eventType == consts.EventTypeOther {
		eventType = event.InferTypeByName(name)
	}
	item := &event.Event{
		OwnerID:       ownerID,
		Name:          name,
		Type:          eventType,
		Host:          consts.EventHostOther,
		DatePrecision: consts.DatePrecisionDay,
	}
	if strings.TrimSpace(row.EventDate) != "" {
		parsed, err := utils.ParseDate(row.EventDate)
		if err != nil {
			return nil, false, fmt.Errorf("事项日期不正确")
		}
		normalized := utils.NormalizeDate(parsed, consts.DatePrecisionDay)
		item.EventDate = utils.DatePtr(normalized)
	}
	if err := s.events.Create(tx, item); err != nil {
		return nil, false, err
	}
	return item, true, nil
}

func (s *TransferSrv) ensureBook(tx *gorm.DB, ownerID uint64, name string) (*book.Book, bool, error) {
	name = strings.TrimSpace(name)
	existing, err := s.books.FindByName(tx, ownerID, name)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	item := &book.Book{
		OwnerID:  ownerID,
		Name:     name,
		Category: consts.DictCodeRenqing,
		Status:   consts.BookStatusNormal,
	}
	if err := s.books.Create(tx, item); err != nil {
		return nil, false, err
	}
	return item, true, nil
}

func (s *TransferSrv) itemKindLabelMap(ownerID uint64) (map[string]string, error) {
	items, err := s.dicts.List(s.rdb.Get(), ownerID, consts.DictTypeItemKind)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(items))
	for i := range items {
		out[items[i].Label] = items[i].Code
	}
	return out, nil
}

func validateRow(row transfer.Row, itemLabels map[string]string) []string {
	errorsList := make([]string, 0)
	if strings.TrimSpace(row.Date) == "" {
		errorsList = append(errorsList, "日期不能为空")
	} else if _, err := utils.ParseDate(row.Date); err != nil {
		errorsList = append(errorsList, "日期格式应为 YYYY-MM-DD")
	}
	if strings.TrimSpace(row.PersonName) == "" {
		errorsList = append(errorsList, "姓名不能为空")
	}
	if strings.TrimSpace(row.EventName) == "" {
		errorsList = append(errorsList, "事项不能为空")
	}
	if strings.TrimSpace(row.BookName) == "" {
		errorsList = append(errorsList, "本子不能为空")
	}
	if directionCode(row.Direction) == "" {
		errorsList = append(errorsList, "方向应为收或送")
	}
	amountCent := int64(0)
	if strings.TrimSpace(row.Amount) != "" {
		parsed, err := utils.ParseYuanToCent(row.Amount)
		if err != nil {
			errorsList = append(errorsList, "现金金额不正确")
		} else {
			amountCent = parsed
		}
	}
	if strings.TrimSpace(row.GiftValue) != "" {
		if _, err := utils.ParseYuanToCent(row.GiftValue); err != nil {
			errorsList = append(errorsList, "礼品估价不正确")
		}
	}
	if amountCent == 0 && strings.TrimSpace(row.GiftName) == "" {
		errorsList = append(errorsList, "金额和礼品至少填一项")
	}
	if strings.TrimSpace(row.ItemKind) != "" && itemKindCode(row.ItemKind, itemLabels) == "" {
		errorsList = append(errorsList, "名目不存在")
	}
	if strings.TrimSpace(row.Status) != "" && statusCode(row.Status, directionCode(row.Direction)) == "" {
		errorsList = append(errorsList, "状态应为待还、已还或无需还")
	}
	return errorsList
}

func cell(line []string, index map[string]int, key string) string {
	position, ok := index[key]
	if !ok || position >= len(line) {
		return ""
	}
	return strings.TrimSpace(line[position])
}

func isEmptyRow(row transfer.Row) bool {
	return strings.TrimSpace(row.Date) == "" && strings.TrimSpace(row.PersonName) == "" &&
		strings.TrimSpace(row.EventName) == "" && strings.TrimSpace(row.BookName) == "" &&
		strings.TrimSpace(row.Amount) == "" && strings.TrimSpace(row.GiftName) == ""
}

func relationLabel(code string) string {
	if label, ok := consts.RelationLabels[code]; ok {
		return label
	}
	return consts.RelationLabels[consts.RelationOther]
}

func relationCode(text string) string {
	text = strings.TrimSpace(text)
	for code, label := range consts.RelationLabels {
		if text == code || text == label {
			return code
		}
	}
	return consts.RelationOther
}

func directionLabel(code string) string {
	if label, ok := consts.DirectionLabels[code]; ok {
		return label
	}
	return code
}

func directionCode(text string) string {
	switch strings.TrimSpace(text) {
	case "收", consts.DirectionIn:
		return consts.DirectionIn
	case "送", consts.DirectionOut:
		return consts.DirectionOut
	default:
		return ""
	}
}

func statusCode(text string, direction string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return consts.RecordStatusPending
	}
	switch text {
	case consts.RecordStatusPending, "未回礼", "对方未回", "待还":
		return consts.RecordStatusPending
	case consts.RecordStatusRepaid, "已回礼", "对方已回", "已还":
		return consts.RecordStatusRepaid
	case consts.RecordStatusNone, "不用回", "无需还", "回礼":
		return consts.RecordStatusNone
	}
	return ""
}

func eventTypeLabel(code string) string {
	if label, ok := consts.EventTypeLabels[code]; ok {
		return label
	}
	return consts.EventTypeLabels[consts.EventTypeOther]
}

func eventTypeCode(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for code, label := range consts.EventTypeLabels {
		if text == code || text == label {
			return code
		}
	}
	return ""
}

func itemKindCode(text string, labels map[string]string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if text == consts.ItemKindCash || text == consts.ItemKindHuopao || text == consts.ItemKindGift {
		return text
	}
	if code, ok := labels[text]; ok {
		return code
	}
	return ""
}

func dedupe(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok || id == 0 {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
