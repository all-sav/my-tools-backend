package merge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mergenator/internal/client/gitlab"
	"mergenator/internal/config"
	"mergenator/internal/service/websocket"
	"mergenator/internal/utils"
)

type service struct {
	cfg       *config.Config
	gitlabCli gitlab.GitLabClient
	wsService websocket.WebSocketService
}

func NewMergeService(cfg *config.Config, gitlabCli gitlab.GitLabClient, wsService websocket.WebSocketService) MergeService {
	return &service{
		cfg:       cfg,
		gitlabCli: gitlabCli,
		wsService: wsService,
	}
}

func (s *service) CreateMR(ctx context.Context, sourceBranch, repoType string, userID int) (string, error) {
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
		time.Sleep(3 * time.Second)
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
	return mrURL, nil
}

func (s *service) sendMsg(userID int, msg, msgType string) {
	_ = s.wsService.SendMessageToUser(context.Background(), userID, msg, msgType)
}
