package handler

import (
	"dredger/pkg/logger"
	"net/http"
	"time"

	"dredger/service"
	"github.com/gin-gonic/gin"
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

	file, err := req.File.Open()
	if err != nil {
		logger.Logger.Errorf("无法打开文件: %v", err)
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	defer file.Close()

	rows, err := h.svc.ImportData(file, req.ShipName, req.Cover)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errBadRequest, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(rows))
}

func (h *Handler) GetShiftStats(c *gin.Context) {
	var query commonRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	start, _ := time.ParseInLocation(time.DateOnly, query.StartDate, time.Local)
	end, _ := time.ParseInLocation(time.DateOnly, query.EndDate, time.Local)
	stats, err := h.svc.GetShiftStats(query.ShipName, start.UnixMilli(), end.UnixMilli())
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

	start, _ := time.ParseInLocation(time.DateOnly, query.StartDate, time.Local)
	end, _ := time.ParseInLocation(time.DateOnly, query.EndDate, time.Local)

	result, err := h.svc.GetOptimalShift(query.ShipName, start.UnixMilli(), end.UnixMilli())
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
	columns := h.svc.GetColumns()
	c.JSON(http.StatusOK, success(columns))
}

func (h *Handler) GetShiftPie(c *gin.Context) {
	var query getShiftPieRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Logger.Errorf("请求参数有误: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	start, _ := time.ParseInLocation(time.DateOnly, query.StartDate, time.Local)
	end, _ := time.ParseInLocation(time.DateOnly, query.EndDate, time.Local)
	pie, err := h.svc.GetShiftPie(query.ShipName, start.UnixMilli(), end.UnixMilli())
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

	start, _ := time.ParseInLocation(time.DateOnly, query.StartDate, time.Local)
	end, _ := time.ParseInLocation(time.DateOnly, query.EndDate, time.Local)
	dataList, err := h.svc.GetColumnDataList(uri.ColumnName, query.ShipName, start.UnixMilli(), end.UnixMilli())
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(dataList))
}
