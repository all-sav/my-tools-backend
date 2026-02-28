package redis

import (
	"context"
	"time"
)

type SessionRepository interface {
	StoreUserSession(ctx context.Context, token, username string, userID int, ttl time.Duration) error
	GetUsernameByToken(ctx context.Context, token string) (string, error)
	GetGitLabUserID(ctx context.Context, username string) (int, error)
	UserExists(ctx context.Context, username string) (bool, error)
	DeleteUserSession(ctx context.Context, token string) error

	StoreWebSocketID(ctx context.Context, userID int, clientID string, ttl time.Duration) error
	GetWebSocketID(ctx context.Context, userID int) (string, error)
	DeleteWebSocketID(ctx context.Context, userID int) error
}
