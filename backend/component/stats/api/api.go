package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/stats/service"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type StatsAPI struct {
	srv *service.StatsSrv
}

func NewStatsAPI(srv *service.StatsSrv) *StatsAPI {
	return &StatsAPI{srv: srv}
}

func (a *StatsAPI) Years(c *gin.Context) {
	years, err := a.srv.Years(c.Request.Context())
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(years))
}

func (a *StatsAPI) Year(c *gin.Context) {
	year := utils.GetInt64FromQuery(c, "year")
	result, err := a.srv.Year(c.Request.Context(), year)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *StatsAPI) Event(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少事项 id", "missing id"))
		return
	}
	result, err := a.srv.Event(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *StatsAPI) Person(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少亲友 id", "missing id"))
		return
	}
	result, err := a.srv.Person(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *StatsAPI) Book(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少礼簿 id", "missing id"))
		return
	}
	result, err := a.srv.Book(c.Request.Context(), uint64(id))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func RegisterRoutes(group *gin.RouterGroup, api *StatsAPI) {
	group.GET("/stats/years", api.Years)
	group.GET("/stats/year", api.Year)
	group.GET("/stats/event", api.Event)
	group.GET("/stats/person", api.Person)
	group.GET("/stats/book", api.Book)
}
