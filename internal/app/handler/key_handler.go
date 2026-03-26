package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/danielwang/redis-manage/internal/app/service"
	"github.com/danielwang/redis-manage/pkg/api"
)

type KeyHandler struct {
	svc *service.KeyService
}

func NewKeyHandler(svc *service.KeyService) *KeyHandler {
	return &KeyHandler{svc: svc}
}

func (h *KeyHandler) ServerInfo(c *gin.Context) {
	info, err := h.svc.ServerInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *KeyHandler) ScanKeys(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")
	cursor, _ := strconv.ParseUint(c.DefaultQuery("cursor", "0"), 10, 64)
	count, _ := strconv.ParseInt(c.DefaultQuery("count", "50"), 10, 64)

	result, err := h.svc.ScanKeys(c.Request.Context(), pattern, cursor, count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *KeyHandler) GetKeyDetail(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	detail, err := h.svc.GetKeyDetail(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *KeyHandler) DeleteKey(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	if err := h.svc.DeleteKey(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *KeyHandler) SetTTL(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	var req struct {
		TTL int64 `json:"ttl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	ttl := time.Duration(req.TTL) * time.Second
	if err := h.svc.SetKeyTTL(c.Request.Context(), key, ttl); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ttl updated"})
}

func (h *KeyHandler) SetStringValue(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "key parameter required"})
		return
	}

	var req struct {
		Value string `json:"value"`
		TTL   int64  `json:"ttl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	ttl := time.Duration(req.TTL) * time.Second
	if err := h.svc.SetStringValue(c.Request.Context(), key, req.Value, ttl); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "value updated"})
}
