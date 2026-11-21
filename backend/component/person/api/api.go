package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/person/service"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/person"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type PersonAPI struct {
	personSrv *service.PersonSrv
	log       *logger.Logger
}

func NewPersonAPI(personSrv *service.PersonSrv) *PersonAPI {
	api := PersonAPI{
		personSrv: personSrv,
		log: logger.New(
			logger.WithModule("person"),
			logger.WithSubModule("api"),
		),
	}
	return &api
}

func (api *PersonAPI) SearchPerson(ctx *gin.Context) {
	param := &person.SearchPersonParam{
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
		Keyword: utils.GetStrFromQuery(ctx, "keyword"),
		Filter:  utils.GetFilter(ctx),
	}
	items, total, err := api.personSrv.SearchPerson(ctx.Request.Context(), param)
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

func (api *PersonAPI) GetPerson(ctx *gin.Context) {
	param := &person.GetPersonParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.personSrv.GetPerson(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *PersonAPI) CreatePerson(ctx *gin.Context) {
	param := &person.CreatePersonParam{}
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
	item, err := api.personSrv.CreatePerson(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *PersonAPI) UpdatePerson(ctx *gin.Context) {
	param := &person.UpdatePersonParam{}
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
	item, err := api.personSrv.UpdatePerson(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *PersonAPI) DeletePerson(ctx *gin.Context) {
	param := &person.GetPersonParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	if err := api.personSrv.DeletePerson(ctx.Request.Context(), param); err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx)
}

func RegisterRoutes(group *gin.RouterGroup, personAPI *PersonAPI) {
	group.GET("/persons", personAPI.SearchPerson)
	group.GET("/persons/detail", personAPI.GetPerson)
	group.POST("/persons", personAPI.CreatePerson)
	group.PUT("/persons", personAPI.UpdatePerson)
	group.DELETE("/persons", personAPI.DeletePerson)
}
