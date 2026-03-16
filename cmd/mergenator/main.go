package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"mergenator/internal/client/gitlab"
	"mergenator/internal/config"
	"mergenator/internal/handler/auth"
	"mergenator/internal/handler/merge"
	settingshandler "mergenator/internal/handler/settings"
	"mergenator/internal/handler/webhook"
	"mergenator/internal/handler/ws"
	"mergenator/internal/infr/logger"
	"mergenator/internal/middleware"
	rd "mergenator/internal/repository/redis"
	"mergenator/internal/repository/settings/dbrepo"
	authSvc "mergenator/internal/service/auth"
	gitlabSvc "mergenator/internal/service/gitlab"
	settingsSvc "mergenator/internal/service/settings"
	wsSvc "mergenator/internal/service/websocket"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// логгер
	logCloser, err := logger.Init(cfg)
	if err != nil {
		log.Fatal("Failed to init logger:", err)
	}

	log := logger.Get()
	log.Info().Msg("Starting MyTools application")

	// Redis клиент
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal().Err(err).Msg("Redis connection failed")
	}
	log.Info().Msg("Connected to Redis")

	// GitLab клиент
	gitlabClient := gitlab.NewClient(cfg.GitLabAPIURL, cfg.GitLabAccessToken)

	// Репозитории
	sessionRepo := rd.NewSessionRepository(rdb)
	settingsRepo, err := dbrepo.NewSettingsDBRepository()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get settings")
	}

	// Сервисы
	wsService := wsSvc.NewWebSocketService([]string{cfg.WSAllowedOrigin}, sessionRepo, cfg.TokenTTL)
	authService := authSvc.NewAuthService(cfg, sessionRepo, gitlabClient)
	settingsService := settingsSvc.NewSettingsService(settingsRepo)
	gitlabService := gitlabSvc.NewGitlabService(gitlabClient, wsService, settingsService)

	// Хендлеры
	authHandler := auth.NewHandler(authService)
	mergeHandler := merge.NewHandler(gitlabService, wsService)
	webhookHandler := webhook.NewHandler(gitlabService, cfg.GitLabWebhookToken)
	wsHandler := ws.NewHandler(wsService)
	settingsHandler := settingshandler.NewHandler(settingsService)

	// Роутер
	router := gin.Default()
	router.Use(middleware.LoggerMiddleware(log))

	// Публичные роуты
	router.POST("/auth/login", authHandler.Login)
	router.POST("/webhook/on-push", webhookHandler.Handle)
	router.GET("/ws", wsHandler.Handle)

	// Приватные роуты
	authGroup := router.Group("/")
	authGroup.Use(middleware.AuthMiddleware(authService))
	{
		authGroup.POST("/merge", mergeHandler.Merge)
		authGroup.POST("/auth/logout", authHandler.Logout)

		authGroup.GET("/settings", settingsHandler.Get)
		authGroup.POST("/settings", settingsHandler.Save)
	}

	startServer(router, cfg)

	defer logCloser.Close()
}

func startServer(router *gin.Engine, cfg *config.Config) {
	log := logger.Get()
	if cfg.OverProxy {
		if err := router.Run("localhost:" + cfg.HTTPPort); err != nil {
			log.Fatal().Err(err)
		}
	} else {
		server := &http.Server{
			Addr:      "localhost:" + cfg.HTTPPort,
			Handler:   router,
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		}
		log.Info().Msg(fmt.Sprintf("HTTPS server running on https://localhost:%s", cfg.HTTPPort))
		if err := server.ListenAndServeTLS(cfg.SSLCertPem, cfg.SSLKeyPem); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err)
		}
	}
}
