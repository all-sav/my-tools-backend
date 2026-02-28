package auth

import (
	"context"
)

type AuthService interface {
	Login(ctx context.Context, username, password, gitlabUser string) (token string, userID int, err error)
	Logout(ctx context.Context, token string) error
	ValidateToken(ctx context.Context, token string) (username string, userID int, err error)
}
