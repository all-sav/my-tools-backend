package db

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"os"
	"time"
)

var (
	RedisClient *redis.Client
	Ctx         = context.Background()
)

const (
	RKeyAuthTokenToGitlabUsername string = "aTokenToGlUsName:"
	RKeyGitLabUserNameToId        string = "glUsName_ID:"
	RKeyGitLabUserIDToWebsocketID string = "glUsID_WSID:"
)

func InitRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0 // todo вынести в env

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Проверяем подключение
	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis successfully")
}

// Сохраняем токен и связку username -> userID
func StoreUserSession(token string, gitlabUsername string, gitlabUserID int, ttl time.Duration) error {
	// Сохраняем токен -> username для быстрой валидации
	err := RedisClient.Set(Ctx, RKeyAuthTokenToGitlabUsername+token, gitlabUsername, ttl).Err()
	if err != nil {
		return err
	}

	// Сохраняем gitlab_username -> userID для быстрого доступа
	err = RedisClient.Set(Ctx, RKeyGitLabUserNameToId+gitlabUsername, gitlabUserID, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

// Получаем username по токену
func GetUsernameByToken(token string) (string, error) {
	return RedisClient.Get(Ctx, RKeyAuthTokenToGitlabUsername+token).Result()
}

// Получаем gitlab userID по username
func GetGitLabUserID(username string) (int, error) {
	val, err := RedisClient.Get(Ctx, RKeyGitLabUserNameToId+username).Int()
	return val, err
}

// Проверяем существует ли пользователь в Redis
func UserExists(username string) (bool, error) {
	exists, err := RedisClient.Exists(Ctx, RKeyGitLabUserNameToId+username).Result()
	return exists == 1, err
}

// Удаляем сессию при выходе
func DeleteUserSession(token string) error {
	// Удаляем токен
	err := RedisClient.Del(Ctx, RKeyAuthTokenToGitlabUsername+token).Err()
	if err != nil {
		return err
	}

	// Не удаляем user:username, чтобы при следующем входе не ходить в GitLab
	return nil
}
