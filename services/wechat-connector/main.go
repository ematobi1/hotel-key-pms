package main

import (
	"context"
	"crypto/sha1"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// WeChatMessage is the canonical inbound message envelope from WeChat.
type WeChatMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgID        int64    `xml:"MsgId"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
}

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	token := mustEnv("WECHAT_TOKEN")
	// appID := mustEnv("WECHAT_APP_ID")         // used when calling WeChat APIs
	// appSecret := mustEnv("WECHAT_APP_SECRET") // keep in Secrets Manager

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// WeChat server verification handshake (GET) and message receive (POST)
	r.GET("/wechat/callback", verifyHandler(token, log))
	r.POST("/wechat/callback", messageHandler(log))

	// Internal API used by other services to send WeChat template messages
	v1 := r.Group("/v1/wechat")
	{
		v1.POST("/template-message", func(c *gin.Context) {
			// TODO: call WeChat template message API
			// POST https://api.weixin.qq.com/cgi-bin/message/template/send
			c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED"}})
		})
		v1.POST("/miniprogram/subscribe-message", func(c *gin.Context) {
			// TODO: send WeChat Mini Program subscribe message
			c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED"}})
		})
	}

	port := getEnvOrDefault("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	go func() {
		log.Info("wechat-connector service listening", zap.String("port", port))
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

func verifyHandler(token string, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.Query("signature")
		timestamp := c.Query("timestamp")
		nonce := c.Query("nonce")
		echostr := c.Query("echostr")

		if verifySignature(token, timestamp, nonce, signature) {
			c.String(http.StatusOK, echostr)
			return
		}
		log.Warn("WeChat signature verification failed")
		c.Status(http.StatusForbidden)
	}
}

func messageHandler(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var msg WeChatMessage
		if err := c.ShouldBindXML(&msg); err != nil {
			log.Warn("failed to parse WeChat message", zap.Error(err))
			c.Status(http.StatusBadRequest)
			return
		}

		log.Info("WeChat message received",
			zap.String("from", msg.FromUserName),
			zap.String("type", msg.MsgType),
			zap.String("event", msg.Event),
		)

		// TODO: route events to booking/notification flows
		// e.g. MsgType=event + Event=CLICK → menu action
		//      MsgType=text → NLP intent routing

		c.String(http.StatusOK, "success")
	}
}

func verifySignature(token, timestamp, nonce, signature string) bool {
	parts := []string{token, timestamp, nonce}
	sort.Strings(parts)
	h := sha1.New()
	fmt.Fprint(h, strings.Join(parts, ""))
	return fmt.Sprintf("%x", h.Sum(nil)) == signature
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
