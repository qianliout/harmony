package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/book/service"
	"harmony/infra/model/book"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type BookAPI struct {
	srv *service.BookSrv
}

func NewBookAPI(srv *service.BookSrv) *BookAPI {
	return &BookAPI{srv: srv}
}

func (a *BookAPI) List(c *gin.Context) {
	items, err := a.srv.List(c.Request.Context(), utils.GetStrFromQuery(c, "status"))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func (a *BookAPI) Detail(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少礼簿 id", "missing id"))
		return
	}
	item, err := a.srv.Detail(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(item))
}

func (a *BookAPI) Create(c *gin.Context) {
	var param book.CreateParam
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

func (a *BookAPI) Update(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少礼簿 id", "missing id"))
		return
	}
	var param book.UpdateParam
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

func (a *BookAPI) Delete(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少礼簿 id", "missing id"))
		return
	}
	if err := a.srv.Delete(c.Request.Context(), uint64(id)); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *BookAPI) MergePreview(c *gin.Context) {
	var param book.MergeParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	preview, err := a.srv.MergePreview(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(preview))
}

func (a *BookAPI) Merge(c *gin.Context) {
	var param book.MergeParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	log, err := a.srv.Merge(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(log))
}

func (a *BookAPI) MergeLogs(c *gin.Context) {
	items, err := a.srv.MergeLogs(c.Request.Context())
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func RegisterRoutes(group *gin.RouterGroup, api *BookAPI) {
	group.GET("/books", api.List)
	group.GET("/books/detail", api.Detail)
	group.POST("/books", api.Create)
	group.PUT("/books", api.Update)
	group.DELETE("/books", api.Delete)
	group.POST("/books/merge/preview", api.MergePreview)
	group.POST("/books/merge", api.Merge)
	group.GET("/books/merge-logs", api.MergeLogs)
}
