package api

import (
	"github.com/gin-gonic/gin"

	"harmony/component/health/service"
	"harmony/infra/logger"
	"harmony/infra/response"
)

type HealthAPI struct {
	healthSrv *service.HealthSrv
	log       *logger.Logger
}

func NewHealthAPI(healthSrv *service.HealthSrv) *HealthAPI {
	api := HealthAPI{
		healthSrv: healthSrv,
		log: logger.New(
			logger.WithModule("health"),
			logger.WithSubModule("api"),
		),
	}
	return &api
}

func (api *HealthAPI) Check(ctx *gin.Context) {
	status, err := api.healthSrv.Check(ctx.Request.Context())
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(status))
}

func RegisterRoutes(group *gin.RouterGroup, healthAPI *HealthAPI) {
	group.GET("/health", healthAPI.Check)
}
