package service

import (
	"context"

	bookstore "harmony/component/book/store"
	personservice "harmony/component/person/service"
	personstore "harmony/component/person/store"
	"harmony/component/record/store"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/model/book"
	"harmony/infra/model/person"
	"harmony/infra/model/record"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type RecordSrv struct {
	recordDal *store.RecordDao
	personDal *personstore.PersonDao
	bookDal   *bookstore.BookDao
	personSrv *personservice.PersonSrv
	log       *logger.Logger
}

func NewRecordSrv(
	recordDal *store.RecordDao,
	personDal *personstore.PersonDao,
	bookDal *bookstore.BookDao,
	personSrv *personservice.PersonSrv,
) *RecordSrv {
	return &RecordSrv{
		recordDal: recordDal,
		personDal: personDal,
		bookDal:   bookDal,
		personSrv: personSrv,
		log:       logger.New(logger.WithModule("record"), logger.WithSubModule("service")),
	}
}

// recordRelations 是一批记录关联的亲友与礼簿，按 id 索引。
type recordRelations struct {
	persons map[int64]*person.Person
	books   map[int64]*book.Book
}

// SearchRecord 统一承载总账、礼簿详情、亲友详情三种视图。
func (s *RecordSrv) SearchRecord(ctx context.Context, param *record.SearchRecordParam) ([]*record.RecordView, int64, error) {
	result := make([]*record.RecordView, 0)
	rows, total, err := s.recordDal.SearchRecord(ctx, param)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Msg("searchRecord failed")
		return result, 0, err
	}
	relations, err := s.searchRelations(ctx, param.OwnerID, rows)
	if err != nil {
		return result, 0, err
	}
	result = s.toRecordViews(rows, relations)
	return result, total, nil
}

func (s *RecordSrv) GetRecord(ctx context.Context, param *record.GetRecordParam) (*record.RecordView, error) {
	row, err := s.getRecord(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	rows := []*record.Record{row}
	relations, err := s.searchRelations(ctx, param.OwnerID, rows)
	if err != nil {
		return nil, err
	}
	views := s.toRecordViews(rows, relations)
	return views[0], nil
}

// SummaryRecord 返回筛选结果的总收、总送与笔数。
func (s *RecordSrv) SummaryRecord(ctx context.Context, param *record.StatRecordParam) (*record.RecordSummary, error) {
	stats, err := s.recordDal.StatRecord(ctx, param)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Msg("statRecord failed")
		return nil, err
	}
	summary := &record.RecordSummary{}
	for _, stat := range stats {
		summary.Count += stat.RecordCount
		summary.InCent += stat.InCent
		summary.OutCent += stat.OutCent
	}
	return summary, nil
}

// CreateRecord 收礼校验礼簿存在；亲友按 id 或姓名匹配，按姓名找不到时新建。
func (s *RecordSrv) CreateRecord(ctx context.Context, param *record.CreateRecordParam) (*record.RecordView, error) {
	if param.Direction == consts.DirectionIn {
		if err := s.checkBook(ctx, param.OwnerID, param.BookID); err != nil {
			return nil, err
		}
	}
	matchParam := &person.MatchPersonParam{
		OwnerID:      param.OwnerID,
		PersonID:     param.PersonID,
		PersonName:   param.PersonName,
		PersonRemark: param.PersonRemark,
	}
	matched, err := s.personSrv.MatchPerson(ctx, matchParam)
	if err != nil {
		return nil, err
	}
	personID := int64(0)
	if matched != nil {
		personID = matched.ID
	}
	if matched == nil {
		createParam := &person.CreatePersonParam{
			OwnerID: param.OwnerID,
			Name:    param.PersonName,
			Remark:  param.PersonRemark,
		}
		created, err := s.personSrv.CreatePerson(ctx, createParam)
		if err != nil {
			return nil, err
		}
		personID = created.ID
	}

	row := &record.Record{
		OwnerID:    param.OwnerID,
		PersonID:   personID,
		BookID:     param.BookID,
		Direction:  param.Direction,
		AmountCent: param.AmountCent,
		Matter:     param.Matter,
		Note:       param.Note,
		OccurredOn: param.OccurredOn,
	}
	if err := s.recordDal.CreateRecord(ctx, row); err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", param.OwnerID).Int64("personID", personID).Msg("createRecord failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("recordID", row.ID).Int64("personID", personID).Msg("record created")
	view, err := s.GetRecord(ctx, &record.GetRecordParam{ID: row.ID, OwnerID: param.OwnerID})
	if err != nil {
		return nil, err
	}
	return view, nil
}

// UpdateRecord 是完整更新；礼簿与事情随方向派生，因此走 Serialize → Check → ToUpdater。
func (s *RecordSrv) UpdateRecord(ctx context.Context, param *record.UpdateRecordParam) (*record.RecordView, error) {
	before, err := s.getRecord(ctx, param.OwnerID, param.ID)
	if err != nil {
		return nil, err
	}
	matchParam := &person.MatchPersonParam{OwnerID: param.OwnerID, PersonID: param.PersonID}
	if _, err := s.personSrv.MatchPerson(ctx, matchParam); err != nil {
		return nil, err
	}
	if param.Direction == consts.DirectionIn {
		if err := s.checkBook(ctx, param.OwnerID, param.BookID); err != nil {
			return nil, err
		}
	}
	after := &record.Record{
		ID:         before.ID,
		OwnerID:    before.OwnerID,
		PersonID:   param.PersonID,
		BookID:     param.BookID,
		Direction:  param.Direction,
		AmountCent: param.AmountCent,
		Matter:     param.Matter,
		Note:       param.Note,
		OccurredOn: param.OccurredOn,
	}
	after = after.Serialize()
	if err := after.Check(); err != nil {
		return nil, response.NewErr(err)
	}
	if err := s.recordDal.UpdateRecord(ctx, param.ID, after.ToUpdater()); err != nil {
		s.log.Error(ctx).Err(err).Int64("recordID", param.ID).Msg("updateRecord failed")
		return nil, err
	}
	s.log.Info(ctx).Int64("recordID", param.ID).Msg("record updated")
	view, err := s.GetRecord(ctx, &record.GetRecordParam{ID: param.ID, OwnerID: param.OwnerID})
	if err != nil {
		return nil, err
	}
	return view, nil
}

func (s *RecordSrv) DeleteRecord(ctx context.Context, param *record.GetRecordParam) error {
	if _, err := s.getRecord(ctx, param.OwnerID, param.ID); err != nil {
		return err
	}
	if err := s.recordDal.DeleteRecord(ctx, param.ID); err != nil {
		s.log.Error(ctx).Err(err).Int64("recordID", param.ID).Msg("deleteRecord failed")
		return err
	}
	s.log.Info(ctx).Int64("recordID", param.ID).Msg("record deleted")
	return nil
}

func (s *RecordSrv) getRecord(ctx context.Context, ownerID, recordID int64) (*record.Record, error) {
	searchParam := &record.SearchRecordParam{OwnerID: ownerID, RecordIDs: []int64{recordID}}
	rows, _, err := s.recordDal.SearchRecord(ctx, searchParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("recordID", recordID).Msg("search record by id failed")
		return nil, err
	}
	if len(rows) == 0 {
		return nil, response.NotFound("记录不存在", "record not found")
	}
	return rows[0], nil
}

func (s *RecordSrv) checkBook(ctx context.Context, ownerID, bookID int64) error {
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

// searchRelations 批量取出记录关联的亲友与礼簿，避免逐条查询。
func (s *RecordSrv) searchRelations(ctx context.Context, ownerID int64, rows []*record.Record) (*recordRelations, error) {
	relations := &recordRelations{
		persons: make(map[int64]*person.Person),
		books:   make(map[int64]*book.Book),
	}
	if len(rows) == 0 {
		return relations, nil
	}
	personIDs := make([]int64, 0, len(rows))
	bookIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		personIDs = append(personIDs, row.PersonID)
		if row.BookID > 0 {
			bookIDs = append(bookIDs, row.BookID)
		}
	}
	personParam := &person.SearchPersonParam{OwnerID: ownerID, PersonIDs: utils.DedupeSorted(personIDs)}
	persons, _, err := s.personDal.SearchPerson(ctx, personParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", ownerID).Msg("search record persons failed")
		return relations, err
	}
	for _, row := range persons {
		relations.persons[row.ID] = row
	}
	if len(bookIDs) == 0 {
		return relations, nil
	}
	bookParam := &book.SearchBookParam{OwnerID: ownerID, BookIDs: utils.DedupeSorted(bookIDs)}
	books, _, err := s.bookDal.SearchBook(ctx, bookParam)
	if err != nil {
		s.log.Error(ctx).Err(err).Int64("ownerID", ownerID).Msg("search record books failed")
		return relations, err
	}
	for _, row := range books {
		relations.books[row.ID] = row
	}
	return relations, nil
}

func (s *RecordSrv) toRecordViews(rows []*record.Record, relations *recordRelations) []*record.RecordView {
	views := make([]*record.RecordView, 0, len(rows))
	for _, row := range rows {
		view := &record.RecordView{
			ID:         row.ID,
			PersonID:   row.PersonID,
			BookID:     row.BookID,
			Direction:  row.Direction,
			AmountCent: row.AmountCent,
			Matter:     row.Matter,
			Note:       row.Note,
			OccurredOn: row.OccurredOn,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		personRow, ok := relations.persons[row.PersonID]
		if ok {
			view.PersonName = personRow.Name
			view.PersonRemark = personRow.Remark
			view.PersonDisplayName = personRow.DisplayName()
		}
		bookRow, ok := relations.books[row.BookID]
		if ok {
			view.BookName = bookRow.Name
		}
		views = append(views, view)
	}
	return views
}
