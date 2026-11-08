package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/record/service"
	"harmony/infra/model/record"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type RecordAPI struct {
	srv *service.RecordSrv
}

func NewRecordAPI(srv *service.RecordSrv) *RecordAPI {
	return &RecordAPI{srv: srv}
}

func (a *RecordAPI) List(c *gin.Context) {
	q := record.Query{
		BookID:    uint64(utils.GetInt64FromQuery(c, "bookId")),
		OriginBookID: uint64(utils.GetInt64FromQuery(c, "originBookId")),
		PersonID:  uint64(utils.GetInt64FromQuery(c, "personId")),
		EventID:   uint64(utils.GetInt64FromQuery(c, "eventId")),
		ItemKind:  utils.GetStrFromQuery(c, "itemKind"),
		Direction: utils.GetStrFromQuery(c, "direction"),
		Status:    utils.GetStrFromQuery(c, "status"),
		From:      utils.GetStrFromQuery(c, "from"),
		To:        utils.GetStrFromQuery(c, "to"),
		Keyword:   utils.GetStrFromQuery(c, "keyword"),
	}
	f := utils.GetFilter(c)
	items, total, err := a.srv.List(c.Request.Context(), q, f)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c,
		response.WithItems(items),
		response.WithTotal(total),
		response.WithPerPage(f.Limit),
		response.WithIndex(f.Offset),
	)
}

func (a *RecordAPI) Detail(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少记录 id", "missing id"))
		return
	}
	item, err := a.srv.Detail(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(item))
}

func (a *RecordAPI) Create(c *gin.Context) {
	var param record.CreateParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	item, err := a.srv.Create(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(item))
}

func (a *RecordAPI) Update(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少记录 id", "missing id"))
		return
	}
	var param record.UpdateParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	item, err := a.srv.Update(c.Request.Context(), uint64(id), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(item))
}

func (a *RecordAPI) Delete(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少记录 id", "missing id"))
		return
	}
	confirm := utils.GetStrFromQuery(c, "confirm") == "true"
	if err := a.srv.Delete(c.Request.Context(), uint64(id), confirm); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *RecordAPI) Repay(c *gin.Context) {
	var param record.RepayParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	result, err := a.srv.Repay(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *RecordAPI) Unlink(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少记录 id", "missing id"))
		return
	}
	var param record.UnlinkParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	if err := a.srv.Unlink(c.Request.Context(), uint64(id), param.OriginID); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *RecordAPI) MarkRepaid(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少记录 id", "missing id"))
		return
	}
	var param record.MarkRepaidParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	if err := a.srv.MarkRepaid(c.Request.Context(), uint64(id), param); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *RecordAPI) ParsePaperNote(c *gin.Context) {
	var param record.PaperNoteParseParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	result, err := a.srv.ParsePaperNote(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *RecordAPI) PendingNotes(c *gin.Context) {
	bookID := uint64(utils.GetInt64FromQuery(c, "bookId"))
	items, err := a.srv.PendingNotes(c.Request.Context(), bookID, utils.GetStrFromQuery(c, "page"))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func (a *RecordAPI) Pages(c *gin.Context) {
	bookID := uint64(utils.GetInt64FromQuery(c, "bookId"))
	if bookID == 0 {
		response.JsonErr(c, response.BadRequest("缺少礼簿 id", "missing bookId"))
		return
	}
	page := utils.GetStrFromQuery(c, "page")
	if page == "" {
		pages, err := a.srv.Pages(c.Request.Context(), bookID)
		if err != nil {
			response.JsonErr(c, err)
			return
		}
		response.JsonOK(c, response.WithItems(pages))
		return
	}
	detail, err := a.srv.PageDetail(c.Request.Context(), bookID, page)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(detail))
}

func (a *RecordAPI) SetPageCheck(c *gin.Context) {
	bookID := uint64(utils.GetInt64FromQuery(c, "bookId"))
	page := utils.GetStrFromQuery(c, "page")
	var param struct {
		DeclaredCent int64 `json:"declaredCent"`
	}
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	if err := a.srv.SetPageCheck(c.Request.Context(), bookID, page, param.DeclaredCent); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func RegisterRoutes(group *gin.RouterGroup, api *RecordAPI) {
	group.GET("/records", api.List)
	group.GET("/records/detail", api.Detail)
	group.POST("/records", api.Create)
	group.PUT("/records", api.Update)
	group.DELETE("/records", api.Delete)
	group.POST("/records/repay", api.Repay)
	group.POST("/records/unlink", api.Unlink)
	group.POST("/records/mark-repaid", api.MarkRepaid)
	group.POST("/paper-notes/parse", api.ParsePaperNote)
	group.GET("/paper-notes/pending", api.PendingNotes)
	group.GET("/books/pages", api.Pages)
	group.PUT("/books/pages", api.SetPageCheck)
}
