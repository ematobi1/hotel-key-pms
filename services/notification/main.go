package main

import (
	"context"
	"encoding/json"
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

// NotificationPayload is the canonical shape accepted by the notification service.
type NotificationPayload struct {
	TenantID    string            `json:"tenant_id" binding:"required"`
	Channel     string            `json:"channel" binding:"required,oneof=email sms push wechat"`
	Recipient   string            `json:"recipient" binding:"required"`
	TemplateID  string            `json:"template_id" binding:"required"`
	Locale      string            `json:"locale"`
	Variables   map[string]string `json:"variables"`
}

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	// TODO: connect to Kafka consumer for async notification events
	// kafkaBrokers := mustEnv("KAFKA_BROKERS")
	// topic := getEnvOrDefault("KAFKA_TOPIC", "hotel-key.notifications")

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/v1/notifications")
	{
		// Synchronous send (use sparingly — prefer async via Kafka)
		v1.POST("/send", func(c *gin.Context) {
			var payload NotificationPayload
			if err := c.ShouldBindJSON(&payload); err != nil {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "VALIDATION_FAILED", "message": err.Error()}})
				return
			}
			if payload.Locale == "" {
				payload.Locale = "en"
			}
			// TODO: route to provider based on channel:
			//   email  → SendGrid / AWS SES
			//   sms    → Twilio / AWS SNS
			//   push   → FCM / APNs
			//   wechat → WeChat template message via wechat-connector service
			log.Info("notification dispatched",
				zap.String("tenant_id", payload.TenantID),
				zap.String("channel", payload.Channel),
				zap.String("template_id", payload.TemplateID),
			)
			c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
		})

		// Webhook for delivery status callbacks from providers
		v1.POST("/webhook/:provider", func(c *gin.Context) {
			provider := c.Param("provider")
			var body json.RawMessage
			c.ShouldBindJSON(&body)
			log.Info("webhook received", zap.String("provider", provider))
			// TODO: parse delivery status and update notification log
			c.Status(http.StatusOK)
		})
	}

	port := getEnvOrDefault("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	go func() {
		log.Info("notification service listening", zap.String("port", port))
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
