package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	maxScan, _ := strconv.ParseInt(c.DefaultQuery("maxScan", "10000"), 10, 64)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	h.svc.ScanBigKeysStream(ctx, pattern, topN, threshold, maxScan, func(p api.ScanProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	})
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
