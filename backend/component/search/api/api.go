package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/search/service"
	"harmony/infra/model/record"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type SearchAPI struct {
	srv *service.SearchSrv
}

func NewSearchAPI(srv *service.SearchSrv) *SearchAPI {
	return &SearchAPI{srv: srv}
}

func (a *SearchAPI) Search(c *gin.Context) {
	query := utils.GetStrFromQuery(c, "q")
	limit := int(utils.GetInt64FromQuery(c, "limit"))
	result, err := a.srv.Search(c.Request.Context(), query, limit)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *SearchAPI) Records(c *gin.Context) {
	q := record.Query{
		Keyword:      utils.GetStrFromQuery(c, "q"),
		BookCategory: utils.GetStrFromQuery(c, "bookCategory"),
		EventType:    utils.GetStrFromQuery(c, "eventType"),
		From:         utils.GetStrFromQuery(c, "from"),
		To:           utils.GetStrFromQuery(c, "to"),
	}
	page := utils.GetInt64FromQuery(c, "page")
	pageSize := utils.GetInt64FromQuery(c, "pageSize")
	items, total, err := a.srv.Records(c.Request.Context(), q, page, pageSize)
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

func RegisterRoutes(group *gin.RouterGroup, api *SearchAPI) {
	group.GET("/search", api.Search)
	group.GET("/search/records", api.Records)
}
