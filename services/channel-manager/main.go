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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/v1/channel-manager")
	{
		// OTA channel connections (Booking.com, Expedia, Rakuten Travel, etc.)
		v1.GET("/channels", func(c *gin.Context) {
			// TODO: list configured OTA channels per hotel
			c.JSON(http.StatusOK, gin.H{"data": []any{}, "meta": gin.H{"total": 0}})
		})

		// Availability & rate push to OTAs (ARI = Availability, Rates, Inventory)
		v1.POST("/ari/push", func(c *gin.Context) {
			// TODO: translate PMS rates to OTA-specific XML/JSON and push via channel adapter
			c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED"}})
		})

		// Receive booking notification from OTA (webhook)
		v1.POST("/bookings/inbound", func(c *gin.Context) {
			// TODO: parse OTA booking (OTA_HotelResNotif_RQ) and create reservation via pms-core
			log.Info("inbound OTA booking received")
			c.JSON(http.StatusAccepted, gin.H{"status": "received"})
		})

		// Booking.com / Expedia modification and cancellation callbacks
		v1.POST("/bookings/modification", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED"}})
		})
	}

	port := getEnvOrDefault("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	go func() {
		log.Info("channel-manager service listening", zap.String("port", port))
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

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
