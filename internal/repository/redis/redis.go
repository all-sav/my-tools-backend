package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	keyTokenToUsername = "aTokenToGlUsName:"
	keyUsernameToID    = "glUsName_ID:"
	keyUserIDToWS      = "glUsID_WSID:"
)

type redisRepository struct {
	client *redis.Client
}

func NewSessionRepository(client *redis.Client) SessionRepository {
	return &redisRepository{client: client}
}

func (r *redisRepository) StoreUserSession(ctx context.Context, token, username string, userID int, ttl time.Duration) error {
	if err := r.client.Set(ctx, keyTokenToUsername+token, username, ttl).Err(); err != nil {
		return err
	}
	return r.client.Set(ctx, keyUsernameToID+username, userID, ttl).Err()
}

func (r *redisRepository) GetUsernameByToken(ctx context.Context, token string) (string, error) {
	return r.client.Get(ctx, keyTokenToUsername+token).Result()
}

func (r *redisRepository) GetGitLabUserID(ctx context.Context, username string) (int, error) {
	val, err := r.client.Get(ctx, keyUsernameToID+username).Int()
	return val, err
}

func (r *redisRepository) UserExists(ctx context.Context, username string) (bool, error) {
	exists, err := r.client.Exists(ctx, keyUsernameToID+username).Result()
	return exists == 1, err
}

func (r *redisRepository) DeleteUserSession(ctx context.Context, token string) error {
	return r.client.Del(ctx, keyTokenToUsername+token).Err()
}

func (r *redisRepository) StoreWebSocketID(ctx context.Context, userID int, clientID string, ttl time.Duration) error {
	return r.client.Set(ctx, keyUserIDToWS+fmt.Sprint(userID), clientID, ttl).Err()
}

func (r *redisRepository) GetWebSocketID(ctx context.Context, userID int) (string, error) {
	return r.client.Get(ctx, keyUserIDToWS+fmt.Sprint(userID)).Result()
}

func (r *redisRepository) DeleteWebSocketID(ctx context.Context, userID int) error {
	return r.client.Del(ctx, keyUserIDToWS+fmt.Sprint(userID)).Err()
}