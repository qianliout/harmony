package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/book/service"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/book"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type BookAPI struct {
	bookSrv *service.BookSrv
	log     *logger.Logger
}

func NewBookAPI(bookSrv *service.BookSrv) *BookAPI {
	api := BookAPI{
		bookSrv: bookSrv,
		log: logger.New(
			logger.WithModule("book"),
			logger.WithSubModule("api"),
		),
	}
	return &api
}

func (api *BookAPI) SearchBook(ctx *gin.Context) {
	param := &book.SearchBookParam{
		OwnerID:  middleware.OwnerID(ctx.Request.Context()),
		Keyword:  utils.GetStrFromQuery(ctx, "keyword"),
		Category: utils.GetStrFromQuery(ctx, "category"),
		Filter:   utils.GetFilter(ctx),
	}
	items, total, err := api.bookSrv.SearchBook(ctx.Request.Context(), param)
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

func (api *BookAPI) GetBook(ctx *gin.Context) {
	param := &book.GetBookParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.bookSrv.GetBook(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *BookAPI) CreateBook(ctx *gin.Context) {
	param := &book.CreateBookParam{}
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
	item, err := api.bookSrv.CreateBook(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *BookAPI) UpdateBook(ctx *gin.Context) {
	param := &book.UpdateBookParam{}
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
	item, err := api.bookSrv.UpdateBook(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *BookAPI) DeleteBook(ctx *gin.Context) {
	param := &book.GetBookParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	if err := api.bookSrv.DeleteBook(ctx.Request.Context(), param); err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx)
}

func RegisterRoutes(group *gin.RouterGroup, bookAPI *BookAPI) {
	group.GET("/books", bookAPI.SearchBook)
	group.GET("/books/detail", bookAPI.GetBook)
	group.POST("/books", bookAPI.CreateBook)
	group.PUT("/books", bookAPI.UpdateBook)
	group.DELETE("/books", bookAPI.DeleteBook)
}
