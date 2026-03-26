package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/danielwang/redis-manage/internal/app/handler"
	"github.com/danielwang/redis-manage/internal/app/service"
	"github.com/danielwang/redis-manage/internal/config"
	"github.com/danielwang/redis-manage/web"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	logger.Info("connected to redis", zap.String("addr", cfg.RedisAddr))

	keySvc := service.NewKeyService(rdb, logger)
	queueSvc := service.NewQueueService(rdb, logger)
	analysisSvc := service.NewAnalysisService(rdb, logger)

	keyH := handler.NewKeyHandler(keySvc)
	queueH := handler.NewQueueHandler(queueSvc)
	analysisH := handler.NewAnalysisHandler(analysisSvc)

	staticFS := web.StaticFS()
	router := handler.NewRouter(keyH, queueH, analysisH, staticFS)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	go func() {
		logger.Info("server starting", zap.Int("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server forced shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}
