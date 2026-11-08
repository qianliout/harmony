package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/dict/service"
	"harmony/infra/model/dict"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type DictAPI struct {
	srv *service.DictSrv
}

func NewDictAPI(srv *service.DictSrv) *DictAPI {
	return &DictAPI{srv: srv}
}

func (a *DictAPI) List(c *gin.Context) {
	items, err := a.srv.List(c.Request.Context(), utils.GetStrFromQuery(c, "type"))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func (a *DictAPI) Create(c *gin.Context) {
	var param dict.CreateParam
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

func (a *DictAPI) Update(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少字典项 id", "missing id"))
		return
	}
	var param dict.UpdateParam
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

func (a *DictAPI) Delete(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少字典项 id", "missing id"))
		return
	}
	if err := a.srv.Delete(c.Request.Context(), uint64(id)); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func RegisterRoutes(group *gin.RouterGroup, api *DictAPI) {
	group.GET("/dicts", api.List)
	group.POST("/dicts", api.Create)
	group.PUT("/dicts", api.Update)
	group.DELETE("/dicts", api.Delete)
}
