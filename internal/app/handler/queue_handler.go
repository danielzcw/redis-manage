package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/danielwang/redis-manage/internal/app/service"
	"github.com/danielwang/redis-manage/pkg/api"
)

type QueueHandler struct {
	svc *service.QueueService
}

func NewQueueHandler(svc *service.QueueService) *QueueHandler {
	return &QueueHandler{svc: svc}
}

func (h *QueueHandler) ListQueues(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")

	queues, err := h.svc.ListQueues(c.Request.Context(), pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, queues)
}

func (h *QueueHandler) GetQueueDetail(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	detail, err := h.svc.GetQueueDetail(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *QueueHandler) Push(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	var req struct {
		Values []string `json:"values"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.svc.Push(c.Request.Context(), key, req.Values); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pushed"})
}

func (h *QueueHandler) Pop(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	count, _ := strconv.ParseInt(c.DefaultQuery("count", "1"), 10, 64)
	if count <= 0 {
		count = 1
	}

	results, err := h.svc.Pop(c.Request.Context(), key, count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"values": results})
}
