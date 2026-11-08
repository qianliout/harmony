package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/event/service"
	"harmony/infra/model/event"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type EventAPI struct {
	srv *service.EventSrv
}

func NewEventAPI(srv *service.EventSrv) *EventAPI {
	return &EventAPI{srv: srv}
}

func (a *EventAPI) List(c *gin.Context) {
	q := event.Query{
		Type:               utils.GetStrFromQuery(c, "type"),
		Host:               utils.GetStrFromQuery(c, "host"),
		From:               utils.GetStrFromQuery(c, "from"),
		To:                 utils.GetStrFromQuery(c, "to"),
		Keyword:            utils.GetStrFromQuery(c, "keyword"),
		IncludePlaceholder: utils.GetStrFromQuery(c, "includePlaceholder") == "true",
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

func (a *EventAPI) Recent(c *gin.Context) {
	limit := int(utils.GetInt64FromQuery(c, "limit"))
	items, err := a.srv.Recent(c.Request.Context(), limit)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func (a *EventAPI) Detail(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少事项 id", "missing id"))
		return
	}
	item, err := a.srv.Detail(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(item))
}

func (a *EventAPI) Create(c *gin.Context) {
	var param event.CreateParam
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

func (a *EventAPI) Update(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少事项 id", "missing id"))
		return
	}
	var param event.UpdateParam
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

func (a *EventAPI) Delete(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少事项 id", "missing id"))
		return
	}
	if err := a.srv.Delete(c.Request.Context(), uint64(id)); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func RegisterRoutes(group *gin.RouterGroup, api *EventAPI) {
	group.GET("/events", api.List)
	group.GET("/events/recent", api.Recent)
	group.GET("/events/detail", api.Detail)
	group.POST("/events", api.Create)
	group.PUT("/events", api.Update)
	group.DELETE("/events", api.Delete)
}
