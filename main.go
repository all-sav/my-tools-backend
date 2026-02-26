package main

import (
	"crypto/tls"
	"log"
	"mergenator/db"
	"mergenator/internal/config"
	"mergenator/internal/service/auth"
	"mergenator/internal/service/gitlab"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func main() {
	config.SetEnvs()
	db.InitRedis()

	go startHTTPServer()

	select {}
}

func startHTTPServer() {
	router := gin.Default()

	// Публичные роуты (без авторизации)
	router.POST("/auth/login", handleLogin)
	router.POST("/webhook/on-push", handleWebhook)

	router.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	// Приватные роуты (с авторизацией)
	authGroup := router.Group("/")
	authGroup.Use(auth.AuthMiddleware())
	{
		authGroup.POST("/merge", handleMerge)
		authGroup.POST("/auth/logout", handleLogout)
	}

	// Создаём TLS-конфигурацию
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if OverProxy {
		if err := router.Run("localhost" + HttpPort); err != nil {
			panic(err)
		}
	} else {
		server := &http.Server{
			Addr:      "localhost" + HttpPort,
			Handler:   router,
			TLSConfig: tlsConfig,
		}

		log.Printf("HTTPS server running on https://localhost%s", HttpPort)
		err := server.ListenAndServeTLS(SSLCertPem, SSLKeyPem)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server failed to start: %v", err)
		}
	}
}
