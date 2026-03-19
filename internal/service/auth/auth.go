package auth

import (
	"context"
	"errors"

	"mergenator/internal/client/gitlab"
	"mergenator/internal/config"
	"mergenator/internal/infr/logger"
	"mergenator/internal/repository/redis"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type service struct {
	cfg         *config.Config
	sessionRepo redis.SessionRepository
	gitlabCli   gitlab.GitLabClient
	log         *zerolog.Logger
}

func NewAuthService(cfg *config.Config, repo redis.SessionRepository, cli gitlab.GitLabClient) AuthService {
	return &service{
		cfg:         cfg,
		sessionRepo: repo,
		gitlabCli:   cli,
		log:         logger.Get(),
	}
}

func (s *service) Login(ctx context.Context, username, password, gitlabUser string) (string, int, error) {
	// Проверка логина/пароля из конфига
	if username != s.cfg.AuthUsername || password != s.cfg.AuthPassword {
		s.log.Info().Str("username", username).Msg("invalid login credentials")
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

	s.log.Info().Str("username", gitlabUser).Msg("user auth success")
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
