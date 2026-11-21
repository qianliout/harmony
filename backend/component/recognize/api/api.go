package api

import (
	"io"

	"github.com/gin-gonic/gin"

	"harmony/component/recognize/service"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/middleware"
	"harmony/infra/model/recognize"
	"harmony/infra/response"
	"harmony/infra/utils"
)

type RecognizeAPI struct {
	recognizeSrv *service.RecognizeSrv
	log          *logger.Logger
}

func NewRecognizeAPI(recognizeSrv *service.RecognizeSrv) *RecognizeAPI {
	api := RecognizeAPI{
		recognizeSrv: recognizeSrv,
		log: logger.New(
			logger.WithModule("recognize"),
			logger.WithSubModule("api"),
		),
	}
	return &api
}

// CreateRecognizeTask 接收 multipart 表单里的 image 文件；文件上传无法走 JSON body，是 POST 规则的唯一例外。
func (api *RecognizeAPI) CreateRecognizeTask(ctx *gin.Context) {
	header, err := ctx.FormFile("image")
	if err != nil {
		response.JSONError(ctx, response.BadRequest("请选择礼簿照片", "image is required"))
		return
	}
	file, err := header.Open()
	if err != nil {
		response.JSONError(ctx, response.BadRequest("读取图片失败", "open image failed"))
		return
	}
	defer func() { _ = file.Close() }()
	// 多读 1 字节，超限时交给 Check 拒绝。
	data, err := io.ReadAll(io.LimitReader(file, consts.RecognizeImageMaxSize+1))
	if err != nil {
		response.JSONError(ctx, response.BadRequest("读取图片失败", "read image failed"))
		return
	}

	param := &recognize.CreateRecognizeTaskParam{
		OwnerID:   middleware.OwnerID(ctx.Request.Context()),
		ImageData: data,
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.recognizeSrv.CreateRecognizeTask(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func (api *RecognizeAPI) GetRecognizeTask(ctx *gin.Context) {
	param := &recognize.GetRecognizeTaskParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	item, err := api.recognizeSrv.GetRecognizeTask(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

// GetRecognizeImage 直接返回原图文件，供核对页对照。
func (api *RecognizeAPI) GetRecognizeImage(ctx *gin.Context) {
	param := &recognize.GetRecognizeTaskParam{
		ID:      utils.GetInt64FromQuery(ctx, "id"),
		OwnerID: middleware.OwnerID(ctx.Request.Context()),
	}
	param = param.Serialize()
	if err := param.Check(); err != nil {
		response.JSONError(ctx, response.NewErr(err))
		return
	}
	path, err := api.recognizeSrv.ResolveRecognizeImage(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "private, max-age=3600")
	ctx.File(path)
}

func (api *RecognizeAPI) CommitRecognizeTask(ctx *gin.Context) {
	param := &recognize.CommitRecognizeTaskParam{}
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
	item, err := api.recognizeSrv.CommitRecognizeTask(ctx.Request.Context(), param)
	if err != nil {
		response.JSONError(ctx, err)
		return
	}
	response.JSONOK(ctx, response.WithItem(item))
}

func RegisterRoutes(group *gin.RouterGroup, recognizeAPI *RecognizeAPI) {
	group.POST("/recognize/tasks", recognizeAPI.CreateRecognizeTask)
	group.GET("/recognize/tasks/detail", recognizeAPI.GetRecognizeTask)
	group.GET("/recognize/tasks/image", recognizeAPI.GetRecognizeImage)
	group.POST("/recognize/tasks/commit", recognizeAPI.CommitRecognizeTask)
}
