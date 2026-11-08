package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/person/service"
	"harmony/infra/model/person"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type PersonAPI struct {
	srv *service.PersonSrv
}

func NewPersonAPI(srv *service.PersonSrv) *PersonAPI {
	return &PersonAPI{srv: srv}
}

func (a *PersonAPI) List(c *gin.Context) {
	q := person.Query{
		Keyword:  utils.GetStrFromQuery(c, "keyword"),
		Relation: utils.GetStrFromQuery(c, "relation"),
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

func (a *PersonAPI) Recent(c *gin.Context) {
	limit := int(utils.GetInt64FromQuery(c, "limit"))
	items, err := a.srv.Recent(c.Request.Context(), limit)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func (a *PersonAPI) Detail(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少亲友 id", "missing id"))
		return
	}
	item, err := a.srv.Detail(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(item))
}

func (a *PersonAPI) Create(c *gin.Context) {
	var param person.CreateParam
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

func (a *PersonAPI) Update(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少亲友 id", "missing id"))
		return
	}
	var param person.UpdateParam
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

func (a *PersonAPI) Delete(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少亲友 id", "missing id"))
		return
	}
	if err := a.srv.Delete(c.Request.Context(), uint64(id)); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func RegisterRoutes(group *gin.RouterGroup, api *PersonAPI) {
	group.GET("/persons", api.List)
	group.GET("/persons/recent", api.Recent)
	group.GET("/persons/detail", api.Detail)
	group.POST("/persons", api.Create)
	group.PUT("/persons", api.Update)
	group.DELETE("/persons", api.Delete)
}
