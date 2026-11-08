package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/auth/service"
	"harmony/infra/middleware"
	"harmony/infra/model/auth"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type AuthAPI struct {
	srv *service.AuthSrv
}

func NewAuthAPI(srv *service.AuthSrv) *AuthAPI {
	return &AuthAPI{srv: srv}
}

func (a *AuthAPI) WxLogin(c *gin.Context) {
	var param auth.WXLoginParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	result, err := a.srv.WxLogin(c.Request.Context(), param.Code)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *AuthAPI) CreateWebTicket(c *gin.Context) {
	result, err := a.srv.CreateWebTicket(c.Request.Context(), c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *AuthAPI) PollWebTicket(c *gin.Context) {
	result, err := a.srv.PollWebTicket(c.Request.Context(), utils.GetStrFromQuery(c, "ticket"))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *AuthAPI) ScanWebTicket(c *gin.Context) {
	if err := a.srv.ScanWebTicket(c.Request.Context(), middleware.OwnerID(c.Request.Context()), utils.GetStrFromQuery(c, "ticket")); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *AuthAPI) ConfirmWebTicket(c *gin.Context) {
	if err := a.srv.ConfirmWebTicket(c.Request.Context(), middleware.OwnerID(c.Request.Context()), utils.GetStrFromQuery(c, "ticket")); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *AuthAPI) CancelWebTicket(c *gin.Context) {
	if err := a.srv.CancelWebTicket(c.Request.Context(), middleware.OwnerID(c.Request.Context()), utils.GetStrFromQuery(c, "ticket")); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *AuthAPI) Me(c *gin.Context) {
	ctx := c.Request.Context()
	result, err := a.srv.Me(ctx, middleware.OwnerID(ctx), middleware.TokenFrom(ctx))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(result))
}

func (a *AuthAPI) Sessions(c *gin.Context) {
	items, err := a.srv.Sessions(c.Request.Context(), middleware.OwnerID(c.Request.Context()))
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItems(items))
}

func (a *AuthAPI) DeleteSession(c *gin.Context) {
	id := utils.GetInt64FromQuery(c, "id")
	if id <= 0 {
		response.JsonErr(c, response.BadRequest("缺少登录设备 id", "missing id"))
		return
	}
	if err := a.srv.RevokeSession(c.Request.Context(), middleware.OwnerID(c.Request.Context()), uint64(id)); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

func (a *AuthAPI) Logout(c *gin.Context) {
	ctx := c.Request.Context()
	if err := a.srv.Logout(ctx, middleware.TokenFrom(ctx)); err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c)
}

// RegisterPublicRoutes 注册无需登录的登录入口。
func RegisterPublicRoutes(group *gin.RouterGroup, api *AuthAPI) {
	group.POST("/auth/wx-login", api.WxLogin)
	group.POST("/auth/web-ticket", api.CreateWebTicket)
	group.GET("/auth/web-ticket", api.PollWebTicket)
}

// RegisterRoutes 注册需要登录的接口。
func RegisterRoutes(group *gin.RouterGroup, api *AuthAPI) {
	group.POST("/auth/web-ticket/scan", api.ScanWebTicket)
	group.POST("/auth/web-ticket/confirm", api.ConfirmWebTicket)
	group.POST("/auth/web-ticket/cancel", api.CancelWebTicket)
	group.GET("/auth/me", api.Me)
	group.GET("/auth/sessions", api.Sessions)
	group.DELETE("/auth/sessions", api.DeleteSession)
	group.POST("/auth/logout", api.Logout)
}
