package main

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"os"
	"time"
)

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

const (
	rKeyAuthTokenToGitlabUsername string = "aTokenToGlUsName:"
	rKeyGitLabUserNameToId        string = "glUsName_ID:"
	rKeyGitLabUserIDToWebsocketID string = "glUsID_WSID:"
)

func initRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0 // todo вынести в env

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Проверяем подключение
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis successfully")
}

// Сохраняем токен и связку username -> userID
func storeUserSession(token string, gitlabUsername string, gitlabUserID int, ttl time.Duration) error {
	// Сохраняем токен -> username для быстрой валидации
	err := redisClient.Set(ctx, rKeyAuthTokenToGitlabUsername+token, gitlabUsername, ttl).Err()
	if err != nil {
		return err
	}

	// Сохраняем gitlab_username -> userID для быстрого доступа
	err = redisClient.Set(ctx, rKeyGitLabUserNameToId+gitlabUsername, gitlabUserID, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

// Получаем username по токену
func getUsernameByToken(token string) (string, error) {
	return redisClient.Get(ctx, rKeyAuthTokenToGitlabUsername+token).Result()
}

// Получаем gitlab userID по username
func getGitLabUserID(username string) (int, error) {
	val, err := redisClient.Get(ctx, rKeyGitLabUserNameToId+username).Int()
	return val, err
}

// Проверяем существует ли пользователь в Redis
func userExists(username string) (bool, error) {
	exists, err := redisClient.Exists(ctx, rKeyGitLabUserNameToId+username).Result()
	return exists == 1, err
}

// Удаляем сессию при выходе
func deleteUserSession(token string) error {
	// Удаляем токен
	err := redisClient.Del(ctx, rKeyAuthTokenToGitlabUsername+token).Err()
	if err != nil {
		return err
	}

	// Не удаляем user:username, чтобы при следующем входе не ходить в GitLab
	return nil
}
