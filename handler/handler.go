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
	fh, err := c.FormFile("file")
	if err != nil {
		logger.Logger.Errorf("获取上传的文件失败: %v", err)
		c.JSON(http.StatusBadRequest, fail(errBadRequest, err.Error()))
		return
	}

	shipName := c.PostForm("shipName")
	if shipName == "" {
		logger.Logger.Warn("shipName为空")
		c.JSON(http.StatusBadRequest, fail(errBadRequest, errBadRequest.String()))
		return
	}

	file, err := fh.Open()
	if err != nil {
		logger.Logger.Errorf("无法打开文件: %v", err)
		c.JSON(http.StatusInternalServerError, fail(errInternalServer, err.Error()))
		return
	}
	defer file.Close()

	rows, err := h.svc.ImportData(file, shipName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fail(errBadRequest, err.Error()))
		return
	}
	c.JSON(http.StatusOK, success(rows))
}

func (h *Handler) GetShiftStats(c *gin.Context) {
	var query struct {
		ShipName  string `form:"ship_name" binding:"required"`
		StartTime string `form:"start_time" binding:"required"`
		EndTime   string `form:"end_time" binding:"required"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start, _ := time.Parse(time.RFC3339, query.StartTime)
	end, _ := time.Parse(time.RFC3339, query.EndTime)

	stats, err := h.svc.GetShiftStats(query.ShipName, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetOptimalShift(c *gin.Context) {
	var query struct {
		ShipName  string `form:"ship_name" binding:"required"`
		StartTime string `form:"start_time" binding:"required"`
		EndTime   string `form:"end_time" binding:"required"`
		Metric    string `form:"metric" binding:"required"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start, _ := time.Parse(time.RFC3339, query.StartTime)
	end, _ := time.Parse(time.RFC3339, query.EndTime)

	result, err := h.svc.AnalyzeOptimalShift(query.ShipName, start, end, query.Metric)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
