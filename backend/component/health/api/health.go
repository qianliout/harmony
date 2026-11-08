package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/health/service"
	"harmony/infra/response"
)

type HealthAPI struct {
	healthSrv *service.HealthSrv
}

func NewHealthAPI(healthSrv *service.HealthSrv) *HealthAPI {
	return &HealthAPI{healthSrv: healthSrv}
}

func (api *HealthAPI) Check(c *gin.Context) {
	status, err := api.healthSrv.Check(c.Request.Context())
	if err != nil {
		response.JsonErr(c, err)
		return
	}
	response.JsonOK(c, response.WithItem(status))
}

func RegisterRoutes(group *gin.RouterGroup, healthAPI *HealthAPI) {
	group.GET("/health", healthAPI.Check)
}
