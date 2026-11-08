package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/view/service"
	"harmony/infra/model/record"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type ViewAPI struct {
	srv *service.ViewSrv
}

func NewViewAPI(srv *service.ViewSrv) *ViewAPI {
	return &ViewAPI{srv: srv}
}

func (a *ViewAPI) PersonTimeline(c *gin.Context) {
	q := record.Query{
		PersonID:  uint64(utils.GetInt64FromQuery(c, "personId")),
		BookID:       uint64(utils.GetInt64FromQuery(c, "bookId")),
		BookCategory: utils.GetStrFromQuery(c, "bookCategory"),
		EventType:    utils.GetStrFromQuery(c, "eventType"),
		EventID:   uint64(utils.GetInt64FromQuery(c, "eventId")),
		ItemKind:  utils.GetStrFromQuery(c, "itemKind"),
		Direction: utils.GetStrFromQuery(c, "direction"),
		Status:    utils.GetStrFromQuery(c, "status"),
		From:      utils.GetStrFromQuery(c, "from"),
		To:        utils.GetStrFromQuery(c, "to"),
		Keyword:   utils.GetStrFromQuery(c, "keyword"),
	}
	timeline, err := a.srv.PersonTimeline(c.Request.Context(), q)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(timeline))
}

func (a *ViewAPI) EventSummary(c *gin.Context) {
	eventID := utils.GetInt64FromQuery(c, "eventId")
	if eventID <= 0 {
		response.JsonErr(c, response.BadRequest("缺少事项 id", "missing eventId"))
		return
	}
	summary, err := a.srv.EventSummary(c.Request.Context(), uint64(eventID))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(summary))
}

func (a *ViewAPI) PendingOwed(c *gin.Context) {
	list, err := a.srv.Pending(c.Request.Context(), "owed")
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(list))
}

func (a *ViewAPI) PendingOwing(c *gin.Context) {
	list, err := a.srv.Pending(c.Request.Context(), "owing")
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(list))
}

func RegisterRoutes(group *gin.RouterGroup, api *ViewAPI) {
	group.GET("/views/person-timeline", api.PersonTimeline)
	group.GET("/views/event-summary", api.EventSummary)
	group.GET("/views/pending/owed", api.PendingOwed)
	group.GET("/views/pending/owing", api.PendingOwing)
}
