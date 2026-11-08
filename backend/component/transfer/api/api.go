package api

import (
	"net/url"

	"github.com/gin-gonic/gin"

	"harmony/component/transfer/service"
	"harmony/infra/model/record"
	"harmony/infra/model/transfer"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type TransferAPI struct {
	srv *service.TransferSrv
}

func NewTransferAPI(srv *service.TransferSrv) *TransferAPI {
	return &TransferAPI{srv: srv}
}

func (a *TransferAPI) Export(c *gin.Context) {
	q := record.Query{
		BookID:    uint64(utils.GetInt64FromQuery(c, "bookId")),
		PersonID:  uint64(utils.GetInt64FromQuery(c, "personId")),
		EventID:   uint64(utils.GetInt64FromQuery(c, "eventId")),
		Direction: utils.GetStrFromQuery(c, "direction"),
		Status:    utils.GetStrFromQuery(c, "status"),
	}
	filename, contentType, payload, err := a.srv.Export(c.Request.Context(), utils.GetStrFromQuery(c, "format"), q)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
	c.Data(200, contentType, payload)
}

func (a *TransferAPI) ImportPreview(c *gin.Context) {
	header, err := c.FormFile("file")
	if err != nil {
		response.JsonErr(c, response.BadRequest("请选择 CSV 文件", "file is required"))
		return
	}
	file, err := header.Open()
	if err != nil {
		response.JsonErr(c, response.BadRequest("读取文件失败", "open file failed"))
		return
	}
	defer func() { _ = file.Close() }()
	preview, err := a.srv.ImportPreview(c.Request.Context(), file)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(preview))
}

func (a *TransferAPI) ImportCommit(c *gin.Context) {
	var param transfer.CommitParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	result, err := a.srv.Commit(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func RegisterRoutes(group *gin.RouterGroup, api *TransferAPI) {
	group.GET("/export", api.Export)
	group.POST("/import/preview", api.ImportPreview)
	group.POST("/import/commit", api.ImportCommit)
}
