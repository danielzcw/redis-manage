package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/danielwang/redis-manage/internal/app/service"
	"github.com/danielwang/redis-manage/pkg/api"
)

type AnalysisHandler struct {
	svc *service.AnalysisService
}

func NewAnalysisHandler(svc *service.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{svc: svc}
}

func (h *AnalysisHandler) ScanBigKeys(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")
	topN, _ := strconv.Atoi(c.DefaultQuery("top", "20"))
	threshold, _ := strconv.ParseInt(c.DefaultQuery("threshold", "0"), 10, 64)

	result, err := h.svc.ScanBigKeys(c.Request.Context(), pattern, topN, threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AnalysisHandler) ScanHotKeys(c *gin.Context) {
	samples, _ := strconv.Atoi(c.DefaultQuery("samples", "200"))

	result, err := h.svc.ScanHotKeys(c.Request.Context(), samples)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
