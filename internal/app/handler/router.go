package handler

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	keyH *KeyHandler,
	queueH *QueueHandler,
	analysisH *AnalysisHandler,
	staticFS fs.FS,
) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Next()
	})

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/info", keyH.ServerInfo)

		apiGroup.GET("/keys", keyH.ScanKeys)
		apiGroup.GET("/keys/detail", keyH.GetKeyDetail)
		apiGroup.DELETE("/keys/detail", keyH.DeleteKey)
		apiGroup.PUT("/keys/ttl", keyH.SetTTL)
		apiGroup.PUT("/keys/value", keyH.SetStringValue)

		apiGroup.GET("/queues", queueH.ListQueues)
		apiGroup.GET("/queues/detail", queueH.GetQueueDetail)
		apiGroup.POST("/queues/push", queueH.Push)
		apiGroup.POST("/queues/pop", queueH.Pop)

		apiGroup.GET("/analysis/bigkeys", analysisH.ScanBigKeys)
		apiGroup.GET("/analysis/hotkeys", analysisH.ScanHotKeys)
	}

	r.NoRoute(gin.WrapH(http.FileServer(http.FS(staticFS))))

	return r
}
