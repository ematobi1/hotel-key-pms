package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/hotel-key/reservation/internal/handler"
	"github.com/hotel-key/reservation/internal/lock"
	"github.com/hotel-key/reservation/internal/repository"
	"github.com/hotel-key/reservation/internal/service"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	pool, err := pgxpool.New(context.Background(), mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	opt, err := redis.ParseURL(mustEnv("REDIS_URL"))
	if err != nil {
		log.Fatal("redis url parse failed", zap.Error(err))
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	locker := lock.NewRedisLocker(rdb, log)
	repo := repository.NewReservationRepository(pool)
	svc := service.NewReservationService(repo, locker, log)
	h := handler.NewReservationHandler(svc, log)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/v1/reservations")
	{
		v1.GET("", h.List)
		v1.POST("", h.Create)
		v1.GET("/:id", h.Get)
		v1.PATCH("/:id/status", h.UpdateStatus)
		v1.POST("/:id/check-in", h.CheckIn)
		v1.POST("/:id/check-out", h.CheckOut)
		v1.DELETE("/:id", h.Cancel)
	}

	port := getEnvOrDefault("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("reservation service listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
