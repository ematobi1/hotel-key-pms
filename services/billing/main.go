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
	"go.uber.org/zap"

	"github.com/hotel-key/billing/internal/appendonly"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	pool, err := pgxpool.New(context.Background(), mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	ledger := appendonly.NewFolioLedger(pool)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/v1/billing")
	{
		v1.GET("/folio/:reservation_id", func(c *gin.Context) {
			tid := c.GetHeader("X-Tenant-ID")
			entries, err := ledger.ListByReservation(c.Request.Context(), tid, c.Param("reservation_id"))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
				return
			}
			balance, currency, _ := ledger.Balance(c.Request.Context(), tid, c.Param("reservation_id"))
			c.JSON(http.StatusOK, gin.H{
				"data":     entries,
				"balance":  balance,
				"currency": currency,
			})
		})

		v1.POST("/folio/:reservation_id/charge", func(c *gin.Context) {
			tid := c.GetHeader("X-Tenant-ID")
			var body struct {
				Description string  `json:"description" binding:"required"`
				Amount      float64 `json:"amount" binding:"required"`
				Currency    string  `json:"currency"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "VALIDATION_FAILED", "message": err.Error()}})
				return
			}
			if body.Currency == "" {
				body.Currency = "JPY"
			}
			entry, err := ledger.Post(c.Request.Context(), appendonly.PostEntryInput{
				TenantID:      tid,
				ReservationID: c.Param("reservation_id"),
				EntryType:     appendonly.EntryCharge,
				Description:   body.Description,
				Amount:        body.Amount,
				Currency:      body.Currency,
				PostedBy:      c.GetHeader("X-User-ID"),
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"data": entry})
		})

		v1.POST("/folio/:reservation_id/payment", func(c *gin.Context) {
			// TODO: integrate payment gateway (Stripe / GMO Payment for Japan)
			c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "payment gateway not configured"}})
		})

		v1.POST("/folio/entries/:entry_id/void", func(c *gin.Context) {
			tid := c.GetHeader("X-Tenant-ID")
			var body struct{ Reason string `json:"reason"` }
			c.ShouldBindJSON(&body)
			entry, err := ledger.Void(c.Request.Context(), tid, c.Param("entry_id"), c.GetHeader("X-User-ID"), body.Reason)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"data": entry})
		})
	}

	port := getEnvOrDefault("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	go func() {
		log.Info("billing service listening", zap.String("port", port))
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
