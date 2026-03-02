package gitlab

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"mergenator/internal/client/gitlab"
	"mergenator/internal/config"
	"mergenator/internal/infr/logger"
	settingsrepo "mergenator/internal/repository/settings"
	"mergenator/internal/service/websocket"
	"mergenator/internal/utils"

	"github.com/rs/zerolog"
)

type service struct {
	cfg          *config.Config
	gitlabCli    gitlab.GitLabClient
	wsService    websocket.WebSocketService
	settingsRepo settingsrepo.SettingsRepository
	log          *zerolog.Logger
}

type MergenatorSettings struct {
	BackendProjectID    string `json:"backend_project_id"`
	FrontendProjectID   string `json:"frontend_project_id"`
	BackendStandBranch  string `json:"backend_stand_branch"`
	FrontendStandBranch string `json:"frontend_stand_branch"`
	CIMainBranch        string `json:"ci_main_branch"`
	RequiredPrefix      string `json:"required_prefix"`
	Prefix              string `json:"prefix"`
	CIPrefix            string `json:"ci_prefix"`
	UpdatedAt           int64  `json:"updated_at"`
}

func NewGitlabService(cfg *config.Config, gitlabCli gitlab.GitLabClient, wsService websocket.WebSocketService, settingsRepo settingsrepo.SettingsRepository) GitlabService {
	return &service{
		cfg:          cfg,
		gitlabCli:    gitlabCli,
		wsService:    wsService,
		settingsRepo: settingsRepo,
		log:          logger.Get(),
	}
}

func (s *service) CreateMR(ctx context.Context, sourceBranch, repoType string, userID int) (string, error) {
	log := s.log.With().
		Str("branch", sourceBranch).
		Str("repo", repoType).
		Int("user_id", userID).
		Logger()

	// data, err := s.settingsRepo.Get("mergenator")

	// var settings MergenatorSettings
	// todo - маппить полученные настройки - из data в MergenatorSettings. И может хранить в памяти чтобы в следующий раз не лазить в репозиторий? а надо ли?

	// Определяем параметры репозитория
	var repo gitlab.Repository
	if repoType == "backend" {
		repo = gitlab.Repository{
			StandBranch: s.cfg.BackendStandBranch,
			ProjectID:   s.cfg.BackendProjectID,
			AssigneeID:  userID,
		}
	} else {
		repo = gitlab.Repository{
			StandBranch: s.cfg.FrontendStandBranch,
			ProjectID:   s.cfg.FrontendProjectID,
			AssigneeID:  userID,
		}
	}

	s.sendMsg(userID, fmt.Sprintf("Запрос на создание MR ветки `%s` [%s]", sourceBranch, repoType), "header")
	s.sendMsg(userID, "Проверка префикса исходной ветки", "default")
	if err := utils.ValidateBranchPrefix(sourceBranch, s.cfg.RequiredPrefix); err != nil {
		return "", err
	}

	s.sendMsg(userID, "Проверка существования исходной ветки", "default")
	exists, err := s.gitlabCli.BranchExists(ctx, sourceBranch, repo.ProjectID)
	if err != nil {
		return "", fmt.Errorf("ошибка проверки ветки: %v", err)
	}
	if !exists {
		return "", fmt.Errorf("ветка %s не найдена", sourceBranch)
	}

	ciBranch := utils.MakeCIBranchName(sourceBranch, s.cfg.Prefix, s.cfg.CIPrefix)

	s.sendMsg(userID, fmt.Sprintf("Проверяем на существование CI-ветки `%s`", ciBranch), "default")
	ciExists, err := s.gitlabCli.BranchExists(ctx, ciBranch, repo.ProjectID)
	if err != nil {
		return "", fmt.Errorf("ошибка проверки CI‑ветки: %v", err)
	}

	s.sendMsg(userID, "Проверяем есть ли уже открытый MR", "default")
	hasMR, _, mrURL, err := s.gitlabCli.HasOpenMR(ctx, ciBranch, repo.StandBranch, repo.ProjectID)
	if err != nil {
		return "", fmt.Errorf("ошибка проверки MR для CI‑ветки: %v", err)
	}
	if hasMR {
		return "", fmt.Errorf("для CI‑ветки %s уже есть открытый MR в %s:\n%s", ciBranch, repo.StandBranch, mrURL)
	}

	if ciExists {
		s.sendMsg(userID, fmt.Sprintf("Удаляем старую CI-ветку `%s`", ciBranch), "default")
		if err := s.gitlabCli.DeleteBranch(ctx, ciBranch, repo); err != nil {
			return "", fmt.Errorf("не удалось удалить CI‑ветку %s: %v", ciBranch, err)
		}
	}

	s.sendMsg(userID, fmt.Sprintf("Создаём новую CI-ветку `%s`", ciBranch), "default")
	if err := s.gitlabCli.CreateBranch(ctx, sourceBranch, ciBranch, repo); err != nil {
		return "", fmt.Errorf("не удалось создать CI‑ветку %s: %v", ciBranch, err)
	}

	hasMR, mrID, _, err := s.gitlabCli.HasOpenMR(ctx, s.cfg.CIMainBranch, ciBranch, repo.ProjectID)
	if err != nil {
		return "", fmt.Errorf("ошибка проверки существующих MR: %v", err)
	}
	if !hasMR {
		mrID, err = s.gitlabCli.MergeBranchInto(ctx, s.cfg.CIMainBranch, ciBranch, repo.ProjectID)
		if err != nil {
			return "", fmt.Errorf("не удалось создать MR: %v", err)
		}
		time.Sleep(6 * time.Second)
	}

	if err := s.gitlabCli.AcceptMR(ctx, mrID, repo.ProjectID); err != nil {
		return "", fmt.Errorf("не удалось принять MR %d: %v", mrID, err)
	}

	s.sendMsg(userID, fmt.Sprintf("Создаём MR от CI-ветки `%s` в ветку стенда `%s`", ciBranch, repo.StandBranch), "default")
	title := strings.TrimPrefix(ciBranch, s.cfg.Prefix+s.cfg.CIPrefix)
	mrURL, err = s.gitlabCli.CreateMR(ctx, ciBranch, title, repo)
	if err != nil {
		return "", err
	}

	s.sendMsg(userID, "MR успешно создан!", "success")
	log.Info().Int("mr_id", mrID).Msg("MR успешно создан")
	return mrURL, nil
}

func (s *service) HandlePush(ctx context.Context, branch, projectID string) error {
	// Проверяем префикс ветки
	if err := utils.ValidateBranchPrefix(branch, s.cfg.RequiredPrefix); err != nil {
		log.Printf("Branch %s skipped: %v", branch, err)
		return nil // Не возвращаем ошибку, просто пропускаем
	}

	ciBranch := utils.MakeCIBranchName(branch, s.cfg.Prefix, s.cfg.CIPrefix)

	// Проверка существования CI-ветки
	log.Printf("Checking CI branch: %s in project %s", ciBranch, projectID)
	ciExists, err := s.gitlabCli.BranchExists(ctx, ciBranch, projectID)
	if err != nil {
		return fmt.Errorf("error checking CI branch: %v", err)
	}
	if !ciExists {
		log.Println("CI branch not found, skipping")
		return nil
	}

	// Получаем stand branch для проекта
	standBranch := utils.GetStandBranchByProjectID(
		projectID,
		s.cfg.BackendStandBranch,
		s.cfg.FrontendStandBranch,
	)

	// Проверяем открытый MR для CI‑ветки
	hasMR, _, _, err := s.gitlabCli.HasOpenMR(ctx, ciBranch, standBranch, projectID)
	if err != nil {
		return fmt.Errorf("error checking MR for CI branch: %v", err)
	}
	if !hasMR {
		log.Println("No open MR found for CI branch")
		return nil
	}

	// Мержим исходную ветку в CI-ветку
	hasMR, mrID, _, err := s.gitlabCli.HasOpenMR(ctx, branch, ciBranch, projectID)
	if err != nil {
		return fmt.Errorf("error checking existing MR: %v", err)
	}

	if !hasMR {
		mrID, err = s.gitlabCli.MergeBranchInto(ctx, branch, ciBranch, projectID)
		if err != nil {
			return fmt.Errorf("failed to create MR: %v", err)
		}
		log.Printf("Created MR #%d, waiting...", mrID)
		time.Sleep(6 * time.Second)
	}

	// Принимаем MR
	if err := s.gitlabCli.AcceptMR(ctx, mrID, projectID); err != nil {
		return fmt.Errorf("failed to accept MR %d: %v", mrID, err)
	}

	log.Printf("Successfully merged %s into %s", branch, ciBranch)
	return nil
}

func (s *service) sendMsg(userID int, msg, msgType string) {
	_ = s.wsService.SendMessageToUser(context.Background(), userID, msg, msgType)
}
