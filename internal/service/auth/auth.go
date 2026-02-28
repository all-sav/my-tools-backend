package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"mergenator/internal/client/gitlab"
	"mergenator/internal/config"
	"mergenator/internal/repository/redis"
)

type service struct {
	cfg         *config.Config
	sessionRepo redis.SessionRepository
	gitlabCli   gitlab.GitLabClient
}

func NewAuthService(cfg *config.Config, repo redis.SessionRepository, cli gitlab.GitLabClient) AuthService {
	return &service{
		cfg:         cfg,
		sessionRepo: repo,
		gitlabCli:   cli,
	}
}

func (s *service) Login(ctx context.Context, username, password, gitlabUser string) (string, int, error) {
	// Проверка логина/пароля из конфига
	if username != s.cfg.AuthUsername || password != s.cfg.AuthPassword {
		return "", 0, errors.New("invalid credentials")
	}

	// Проверка существования пользователя в Redis или GitLab
	exists, err := s.sessionRepo.UserExists(ctx, gitlabUser)
	if err != nil {
		return "", 0, err
	}

	var userID int
	if exists {
		userID, err = s.sessionRepo.GetGitLabUserID(ctx, gitlabUser)
		if err != nil {
			return "", 0, err
		}
	} else {
		userID, err = s.gitlabCli.FindUserID(ctx, gitlabUser)
		if err != nil {
			return "", 0, err
		}
	}

	token := uuid.New().String()
	if err := s.sessionRepo.StoreUserSession(ctx, token, gitlabUser, userID, s.cfg.TokenTTL); err != nil {
		return "", 0, err
	}
	return token, userID, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	return s.sessionRepo.DeleteUserSession(ctx, token)
}

func (s *service) ValidateToken(ctx context.Context, token string) (string, int, error) {
	username, err := s.sessionRepo.GetUsernameByToken(ctx, token)
	if err != nil {
		return "", 0, err
	}
	userID, err := s.sessionRepo.GetGitLabUserID(ctx, username)
	if err != nil {
		return "", 0, err
	}
	return username, userID, nil
}
