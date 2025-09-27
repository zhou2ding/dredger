package handler

import (
	"context"
	"dredger/model"
	"dredger/pkg/logger"
	"dredger/service"
	"fmt"
	"mime"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ImportData(c *gin.Context) {
	var req importDataRequest
	if err := c.ShouldBind(&req); err != nil {
		logger.Logger.Errorf("获取上传的文件失败: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	startDate, endDate, err := parseFileName(req.File.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}
	file, err := req.File.Open()
	if err != nil {
		logger.Logger.Errorf("无法打开文件: %v", err)
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	defer file.Close()

	rows, err := h.svc.ImportData(file, req.ShipName, req.Cover, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errBadRequest, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(rows))

	logger.Logger.Infof("导入 %s 成功！", req.File.Filename)
}

func (h *Handler) GetShiftStats(c *gin.Context) {
	var query commonRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	stats, err := h.svc.GetShiftStats(query.ShipName, query.StartDate, query.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(stats))
}

func (h *Handler) GetOptimalShift(c *gin.Context) {
	var query getOptimalShiftRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	result, err := h.svc.GetOptimalShift(query.ShipName, query.StartDate, query.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(result))
}

func (h *Handler) GetShipList(c *gin.Context) {
	ships, err := h.svc.GetShipList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(ships))
}

func (h *Handler) GetColumns(c *gin.Context) {
	columns := h.svc.GetColumns(c.Param("shipName"))
	c.JSON(http.StatusOK, success(columns))
}

func (h *Handler) GetShiftPie(c *gin.Context) {
	var query getShiftPieRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	pie, err := h.svc.GetShiftPie(query.ShipName, query.StartDate, query.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(pie))
}

func (h *Handler) GetHistoryData(c *gin.Context) {
	var uri getHistoryDataUri
	if err := c.ShouldBindUri(&uri); err != nil {
		logger.Logger.Errorf("路径参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	var query commonRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	dataList, err := h.svc.GetColumnDataList(uri.ColumnName, query.ShipName, query.StartDate, query.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(dataList))
}

func (h *Handler) GetGlobalTimeRange(c *gin.Context) {
	results, err := h.svc.GetGlobalTimeRange()
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(results))
}

func (h *Handler) GetNoneEmptyTimeRange(c *gin.Context) {
	var req commonRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	results, err := h.svc.GetNonEmptyTimeRange(req.ShipName, req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(results))
}

func (h *Handler) SetTheoryOptimal(c *gin.Context) {
	var req setTheoryOptimalRequest
	// 从请求体中绑定 JSON 数据
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Errorf("设置理论最优参数失败，请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	// 将请求数据映射到 model 结构体
	params := &model.TheoryOptimalParam{
		ShipName:                     req.ShipName,
		Flow:                         req.Flow,
		Concentration:                req.Concentration,
		SPumpRpm:                     req.SPumpRpm,
		CutterDepth:                  req.CutterDepth,
		CarriageTravel:               req.CarriageTravel,
		HorizontalSpeed:              req.HorizontalSpeed,
		BoosterPumpDischargePressure: req.BoosterPumpDischargePressure,
		VacuumDegree:                 req.VacuumDegree,
	}

	// 调用 service 层处理业务逻辑
	if err := h.svc.SetTheoryOptimalParams(params); err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	// 成功后返回，data 可以为 nil 或者一个简单的成功提示
	c.JSON(http.StatusOK, success(nil))
}

func (h *Handler) GetTheoryOptimal(c *gin.Context) {
	var req getTheoryOptimalRequest
	// 从 Query 参数中绑定 shipName
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Logger.Errorf("获取理论最优参数失败，请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	params, err := h.svc.GetTheoryOptimalParams(req.ShipName)
	if err != nil {
		// 这里是处理 service 层返回的真正的数据库错误
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	// 如果 params 为 nil (即没找到记录)，success(nil) 会返回一个 data 为 null 的 JSON 对象
	// 前端可以根据 data 是否为 null 来判断
	c.JSON(http.StatusOK, success(params))
}

func (h *Handler) GetAllShiftParameters(c *gin.Context) {
	var query commonRequest // 复用 commonRequest，它包含 ShipName, StartDate, EndDate
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	result, err := h.svc.GetAllShiftParameters(query.ShipName, query.StartDate, query.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(result))
}

func (h *Handler) GenerateSolid(c *gin.Context) {
	var req genSolidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON payload: %v", err)})
		return
	}

	params := service.ExecutionParams{
		Action:                  req.Action,
		GeologyDbFile:           req.GeologyDbFile,
		CalculationBoundaryFile: req.CalculationBoundaryFile,
		DesignDepthFile:         req.DesignDepthFile,
		MudlineFile:             req.MudlineFile,
		ReferenceZ:              req.ReferenceZ,
		GridDistanceXY:          req.GridDistanceXY,
		GridDistanceZ:           req.GridDistanceZ,
		Threshold:               req.Threshold,
		PileX:                   req.PileX,
		PileY:                   req.PileY,
		ProfileX1:               req.ProfileX1,
		ProfileY1:               req.ProfileY1,
		ProfileX2:               req.ProfileX2,
		ProfileY2:               req.ProfileY2,
		SpecifiedX:              req.SpecifiedX,
		SpecifiedY:              req.SpecifiedY,
		SpecifiedLength:         req.SpecifiedLength,
		SpecifiedWidth:          req.SpecifiedWidth,
	}

	result, err := h.svc.ExecuteSolidProgram(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(result))
}

func (h *Handler) RunDemo(c *gin.Context) {
	demoStr := c.Query("demo")
	n, err := strconv.Atoi(demoStr)
	if err != nil || n < 1 || n > 6 {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "invalid demo id"))
		return
	}
	id := service.DemoID(n)

	if err := c.Request.ParseMultipartForm(64 << 20); err != nil { // 64MB
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "invalid multipart form"))
		return
	}
	form := c.Request.MultipartForm

	// 解析表单里非文件字段
	p := &service.DemoParams{
		RefZ:      cast.ToFloat64(c.PostForm("ref_z")),
		GridXY:    cast.ToFloat64(c.PostForm("grid_xy")),
		GridZ:     cast.ToFloat64(c.PostForm("grid_z")),
		CX:        cast.ToFloat64(c.PostForm("cx")),
		CY:        cast.ToFloat64(c.PostForm("cy")),
		Length:    cast.ToFloat64(c.PostForm("length")),
		Width:     cast.ToFloat64(c.PostForm("width")),
		Depth:     cast.ToFloat64(c.PostForm("depth")),
		Height:    cast.ToFloat64(c.PostForm("height")),
		X1:        cast.ToFloat64(c.PostForm("x1")),
		Y1:        cast.ToFloat64(c.PostForm("y1")),
		X2:        cast.ToFloat64(c.PostForm("x2")),
		Y2:        cast.ToFloat64(c.PostForm("y2")),
		Threshold: cast.ToFloat64(c.PostForm("threshold")),
	}

	// 上下文 + 超时（防止卡死）
	ctx, cancel := context.WithTimeout(c, 10*time.Minute)
	defer cancel()

	newFiles, err := h.svc.RunDemo(ctx, id, p, form)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(newFiles))
}

type openFileRequest struct {
	Path string `json:"path"`
}

func (h *Handler) OpenFile(c *gin.Context) {
	var req openFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "bad json"))
		return
	}
	abs, err := filepath.Abs(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "bad path"))
		return
	}
	// 只允许 pys 目录下
	root, _ := filepath.Abs("./pys")
	if !strings.HasPrefix(abs, root) {
		c.JSON(http.StatusForbidden, fail(errBadRequest, "path not allowed"))
		return
	}
	// Windows: 用默认程序打开
	cmd := exec.Command("cmd", "/C", "start", "", abs)
	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(nil))
}

func (h *Handler) ServeFile(c *gin.Context) {
	var req serveReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "missing path"))
		return
	}
	abs, err := filepath.Abs(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "bad path"))
		return
	}

	root, _ := filepath.Abs("./pys")
	if !strings.HasPrefix(abs, root) {
		c.JSON(http.StatusForbidden, fail(errBadRequest, "path not allowed"))
		return
	}

	// 根据扩展名设置 Content-Type（xlsx/ png/ html/ txt）
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(abs))); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.File(abs)
}

func (h *Handler) GetLatestResults(c *gin.Context) {
	// ?ids=1,2,3
	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "missing ids query parameter"))
		return
	}

	idParts := strings.Split(idsStr, ",")
	var demoIDs []service.DemoID
	for _, part := range idParts {
		id, err := strconv.Atoi(part)
		if err != nil {
			c.JSON(http.StatusBadRequest, fail(errBadRequest, "invalid id in ids list"))
			return
		}
		demoIDs = append(demoIDs, service.DemoID(id))
	}

	results, err := h.svc.GetLatestResults(demoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(results))
}

func (h *Handler) OpenLocation(c *gin.Context) {
	var req OpenLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, fail(errBadRequest, "请求参数无效: "+err.Error()))
		return
	}

	if err := h.svc.OpenLocation(req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(nil))
}

func (h *Handler) GetPlaybackData(c *gin.Context) {
	var query getPlaybackDataRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	data, err := h.svc.GetPlaybackData(query.ShipName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}

	c.JSON(http.StatusOK, success(data))
}
