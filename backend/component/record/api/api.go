package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/record/service"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/record"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type RecordAPI struct {
	recordSrv *service.RecordSrv
	log       *logger.Logger
}

func NewRecordAPI(recordSrv *service.RecordSrv) *RecordAPI {
	api := RecordAPI{
		recordSrv: recordSrv,
		log: logger.New(
			logger.WithModule("record"),
			logger.WithSubModule("api"),
		),
	}
	return &api
}

func (api *RecordAPI) SearchRecord(ctx *gin.Context) {
	param := &record.SearchRecordParam{
		OwnerID:   middleware.OwnerID(ctx.Request.Context()),
		PersonID:  utils.GetInt64FromQuery(ctx, "personId"),
		BookID:    utils.GetInt64FromQuery(ctx, "bookId"),
		Direction: utils.GetStrFromQuery(ctx, "direction"),
		From:      utils.GetStrFromQuery(ctx, "from"),
		To:        utils.GetStrFromQuery(ctx, "to"),
		Keyword:   utils.GetStrFromQuery(ctx, "keyword"),
		Filter:    utils.GetFilter(ctx),
	}
	items, total, err := api.recordSrv.SearchRecord(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx,
		response.WithItems(items),
		response.WithTotalItems(total),
		response.WithItemsPerPage(param.Filter.Limit),
		response.WithStartIndex(param.Filter.Offset),
	)
}

// SummaryRecord 接收与 SearchRecord 相同的筛选条件，返回总收、总送与笔数。
func (api *RecordAPI) SummaryRecord(ctx *gin.Context) {
	param := &record.StatRecordParam{
		OwnerID:   middleware.OwnerID(ctx.Request.Context()),
		PersonID:  utils.GetInt64FromQuery(ctx, "personId"),
		BookID:    utils.GetInt64FromQuery(ctx, "bookId"),
		Direction: utils.GetStrFromQuery(ctx, "direction"),
		From:      utils.GetStrFromQuery(ctx, "from"),
		To:        utils.GetStrFromQuery(ctx, "to"),
		Keyword:   utils.GetStrFromQuery(ctx, "keyword"),
	}
	item, err := api.recordSrv.SummaryRecord(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *RecordAPI) GetRecord(ctx *gin.Context) {
	param := &record.GetRecordParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.recordSrv.GetRecord(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *RecordAPI) CreateRecord(ctx *gin.Context) {
	param := &record.CreateRecordParam{}
	if err := ctx.ShouldBindJSON(param); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	param.OwnerID = middleware.OwnerID(ctx.Request.Context())
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.recordSrv.CreateRecord(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *RecordAPI) UpdateRecord(ctx *gin.Context) {
	param := &record.UpdateRecordParam{}
	if err := ctx.ShouldBindJSON(param); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	param.ID = utils.GetInt64FromQuery(ctx, "id")
	param.OwnerID = middleware.OwnerID(ctx.Request.Context())
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.recordSrv.UpdateRecord(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *RecordAPI) DeleteRecord(ctx *gin.Context) {
	param := &record.GetRecordParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	if err := api.recordSrv.DeleteRecord(ctx.Request.Context(), param); err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx)
}

func RegisterRoutes(group *gin.RouterGroup, recordAPI *RecordAPI) {
	group.GET("/records", recordAPI.SearchRecord)
	group.GET("/records/summary", recordAPI.SummaryRecord)
	group.GET("/records/detail", recordAPI.GetRecord)
	group.POST("/records", recordAPI.CreateRecord)
	group.PUT("/records", recordAPI.UpdateRecord)
	group.DELETE("/records", recordAPI.DeleteRecord)
}
