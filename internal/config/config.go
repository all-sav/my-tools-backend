package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort        string
	WSAllowedOrigin string

	GitLabAPIURL       string
	GitLabAccessToken  string
	GitLabWebhookToken string

	OverProxy  bool
	SSLCertPem string
	SSLKeyPem  string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	AuthUsername string
	AuthPassword string
	TokenTTL     time.Duration

	LogFile       string
	LogLevel      string
	LogMaxSize    int
	LogMaxBackups int
	LogMaxAge     int
	LogCompress   bool
	LogConsole    bool
	LogJSON       bool
}

func Load() (*Config, error) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("файл .env не найден, юзаем дефолтные данные")
	}

	ttl, _ := time.ParseDuration(os.Getenv("TOKEN_TTL"))
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	maxSize, _ := strconv.Atoi(getEnv("LOG_MAX_SIZE", "100"))
	maxBackups, _ := strconv.Atoi(getEnv("LOG_MAX_BACKUPS", "3"))
	maxAge, _ := strconv.Atoi(getEnv("LOG_MAX_AGE", "28"))
	compress := getEnv("LOG_COMPRESS", "true") == "true"
	console := getEnv("LOG_CONSOLE", "true") == "true"
	jsonLog := getEnv("LOG_JSON", "false") == "true"

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	return &Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		WSAllowedOrigin: getEnv("APP_URL", ""),

		GitLabAPIURL:       getEnv("GITLAB_API_URL", ""),
		GitLabAccessToken:  getEnv("GITLAB_ACCESS_TOKEN", ""),
		GitLabWebhookToken: getEnv("GITLAB_WEBHOOK_TOKEN", ""),

		OverProxy:  os.Getenv("OVER_PROXY") == "true",
		SSLCertPem: getEnv("SSL_CERT_PEM", ""),
		SSLKeyPem:  getEnv("SSL_KEY_PEM", ""),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,

		AuthUsername: getEnv("AUTH_USERNAME", ""),
		AuthPassword: getEnv("AUTH_PASSWORD", ""),
		TokenTTL:     ttl,

		LogFile:       getEnv("LOG_FILE", "./logs/mergenator.log"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LogMaxSize:    maxSize,
		LogMaxBackups: maxBackups,
		LogMaxAge:     maxAge,
		LogCompress:   compress,
		LogConsole:    console,
		LogJSON:       jsonLog,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
