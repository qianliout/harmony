package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/recognize/service"
	"harmony/infra/model/recognize"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type RecognizeAPI struct {
	srv *service.RecognizeSrv
}

func NewRecognizeAPI(srv *service.RecognizeSrv) *RecognizeAPI {
	return &RecognizeAPI{srv: srv}
}

func (a *RecognizeAPI) CreateTask(c *gin.Context) {
	header, err := c.FormFile("image")
	if err != nil {
		response.JsonErr(c, response.BadRequest("请选择礼簿照片", "image is required"))
		return
	}
	file, err := header.Open()
	if err != nil {
		response.JsonErr(c, response.BadRequest("读取图片失败", "open image failed"))
		return
	}
	defer func() { _ = file.Close() }()
	task, err := a.srv.CreateTask(c.Request.Context(), file)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(task))
}

func (a *RecognizeAPI) Detail(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少识别任务 id", "missing id"))
		return
	}
	task, err := a.srv.Detail(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(task))
}

func (a *RecognizeAPI) List(c *gin.Context) {
	page := utils.GetInt64FromQuery(c, "page")
	pageSize := utils.GetInt64FromQuery(c, "pageSize")
	items, total, err := a.srv.List(c.Request.Context(), page, pageSize)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c,
		response.WithItems(items),
		response.WithTotal(total),
		response.WithPerPage(pageSize),
	)
}

func (a *RecognizeAPI) Commit(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少识别任务 id", "missing id"))
		return
	}
	var param recognize.CommitParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	result, err := a.srv.Commit(c.Request.Context(), uint64(id), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func RegisterRoutes(group *gin.RouterGroup, api *RecognizeAPI) {
	group.POST("/recognize/tasks", api.CreateTask)
	group.GET("/recognize/tasks", api.List)
	group.GET("/recognize/tasks/detail", api.Detail)
	group.POST("/recognize/tasks/commit", api.Commit)
}
