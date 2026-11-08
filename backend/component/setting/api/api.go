package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/setting/service"
	"harmony/infra/model/setting"
	"harmony/infra/response"
)

type SettingAPI struct {
	srv *service.SettingSrv
}

func NewSettingAPI(srv *service.SettingSrv) *SettingAPI {
	return &SettingAPI{srv: srv}
}

func (a *SettingAPI) Get(c *gin.Context) {
	settings, err := a.srv.Get(c.Request.Context())
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(settings))
}

func (a *SettingAPI) Update(c *gin.Context) {
	var param setting.UpdateParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.JsonErr(c, response.BadRequest("提交的数据格式不正确", "invalid request body"))
		return
	}
	settings, err := a.srv.Update(c.Request.Context(), param)
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(settings))
}

func RegisterRoutes(group *gin.RouterGroup, api *SettingAPI) {
	group.GET("/settings", api.Get)
	group.PUT("/settings", api.Update)
}
