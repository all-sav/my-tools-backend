package main

import (
	"fmt"
	"log"
	"mergenator/dto"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MergeRequest struct {
	SourceBranch string `json:"source_branch"`
	Repo         string `json:"repo"`
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	GitLabUser string `json:"gitlab_user"`
}
type LoginResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	GitLabUserID int    `json:"gitlab_user_id,omitempty"`
	GitLabToken  string `json:"gitlab_token,omitempty"`
}

// Логика для кнопки "Создать MR"
func createLogicMR(branch string, repo Repository, gitlabUserId int) (error, string) {
	sendMessageByID(gitlabUserId, fmt.Sprintf(
		"Запрос на создание MR ветки `%s` [%s]",
		branch, getProjectNameByID(repo.ProjectId)), WSMessageTypeHeader)

	// 1. Проверка префикса исходной ветки
	sendMessageByID(gitlabUserId, "Проверка префикса исходной ветки", WSMessageTypeDefault)
	if err := validateBranchPrefix(branch); err != nil {
		return err, ""
	}

	// 2. Проверка существования исходной ветки
	sendMessageByID(gitlabUserId, "Проверка существования исходной ветки", WSMessageTypeDefault)
	exists, err := branchExistsInRepo(branch, repo.ProjectId)
	if err != nil {
		return fmt.Errorf("ошибка проверки ветки: %v", err), ""
	}
	if !exists {
		return fmt.Errorf("ветка %s не найдена", branch), ""
	}

	// 3. Формирование имени CI‑ветки
	ciBranch := makeCIBranchName(branch)

	// 4. Проверка существования CI‑ветки
	sendMessageByID(gitlabUserId, fmt.Sprintf("Проверяем на существование CI-ветки `%s`", ciBranch), WSMessageTypeDefault)
	ciExists, err := branchExistsInRepo(ciBranch, repo.ProjectId)
	if err != nil {
		return fmt.Errorf("Ошибка проверки CI‑ветки: %v", err), ""
	}

	// 5. Проверка открытого MR для CI‑ветки
	sendMessageByID(gitlabUserId, "Проверяем есть ли уже открытый MR", WSMessageTypeDefault)
	hasMR, mrID, mrUrl, err := hasOpenMR(ciBranch, repo.StandBranch, repo.ProjectId)
	if err != nil {
		return fmt.Errorf("Ошибка проверки MR для CI‑ветки: %v", err), ""
	}
	if hasMR {
		return fmt.Errorf("Для CI‑ветки %s уже есть открытый MR в %s:\n%s", ciBranch, repo.StandBranch, getLink(Link{Href: mrUrl})), ""
	}

	// 6. Если CI‑ветка существует — удаляем её
	if ciExists {
		sendMessageByID(gitlabUserId, fmt.Sprintf("Удаляем старую CI-ветку `%s`", ciBranch), WSMessageTypeDefault)
		if err := deleteRemoteBranch(ciBranch, repo); err != nil {
			return fmt.Errorf("не удалось удалить CI‑ветку %s: %v", ciBranch, err), ""
		}
	}

	// 7. Создание CI‑ветки от исходной
	sendMessageByID(gitlabUserId, fmt.Sprintf("Создаём новую CI-ветку `%s`", ciBranch), WSMessageTypeDefault)
	if err := createRemoteBranch(branch, ciBranch, repo); err != nil {
		return fmt.Errorf("не удалось создать CI‑ветку %s: %v", ciBranch, err), ""
	}

	// 8. Проверка: есть ли уже открытый MR для этих веток?
	hasMR, mrID, mrUrl, err = hasOpenMR(CIMainBranch, ciBranch, repo.ProjectId)
	if err != nil {
		return fmt.Errorf("ошибка проверки существующих MR: %v", err), ""
	}
	if hasMR {
		// MR уже существует — не создаём новый, а используем существующий
		log.Printf("Уже есть открытый MR №%d для %s → %s", mrID, CIMainBranch, ciBranch)
	} else {
		// Создаём новый MR
		mrID, err = mergeBranchInto(CIMainBranch, ciBranch, repo.ProjectId)
		if err != nil {
			return fmt.Errorf("не удалось создать MR: %v", err), ""
		}
		log.Printf("Засыпаем...")
		time.Sleep(3 * time.Second)
		log.Printf("Просыпаемся...")
	}

	// 9. Принятие MR (фактическое слияние)
	if err := acceptMergeRequest(mrID, repo.ProjectId); err != nil {
		return fmt.Errorf("не удалось принять MR %d: %v", mrID, err), ""
	}

	// 10. Создание MR от CI‑ветки
	sendMessageByID(gitlabUserId, fmt.Sprintf("Создаём MR от CI-ветки `%s` в ветку стенда `%s`", ciBranch, repo.StandBranch), WSMessageTypeDefault)
	title := strings.TrimPrefix(ciBranch, Prefix+CIPrefix)
	mrURL, err := createGitLabMR(ciBranch, title, repo)
	if err != nil {
		return err, ""
	}

	return nil, mrURL
}

func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("Неверный формат запроса"))
		return
	}

	// Проверяем логин/пароль из .env
	validUsername := os.Getenv("AUTH_USERNAME")
	validPassword := os.Getenv("AUTH_PASSWORD")

	if req.Username != validUsername || req.Password != validPassword {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Неверный логин или пароль"))
		return
	}

	// Получаем TTL токена из env
	ttlStr := os.Getenv("TOKEN_TTL")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		ttl = 24 * time.Hour // По умолчанию 24 часа
	}

	var gitlabUserID int

	// Проверяем, есть ли уже пользователь в Redis
	exists, err := userExists(req.GitLabUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Ошибка проверки пользователя"))
		return
	}

	if exists {
		// Если есть, берем ID из Redis
		gitlabUserID, err = getGitLabUserID(req.GitLabUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Ошибка получения данных пользователя"))
			return
		}
	} else {
		// Если нет, ищем в GitLab
		gitlabUserID, err = findGitLabUserID(req.GitLabUser)
		if err != nil {
			c.JSON(http.StatusOK, dto.ErrorResponse("Пользователь GitLab не найден: "+err.Error()))
			return
		}
	}

	// Генерируем уникальный токен
	token := uuid.New().String()

	// Сохраняем сессию в Redis
	err = storeUserSession(token, req.GitLabUser, gitlabUserID, ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Ошибка сохранения сессии"))
		return
	}

	// Сохраняем связку wsClientID с пользователем (опционально)
	// if req.WSClientID != "" {
	// 	// Можно сохранить в Redis: "ws:"+wsClientID -> username
	// 	redisClient.Set(ctx, "ws:"+req.WSClientID, req.GitLabUser, ttl)
	// }

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.LoginResponseData{
		Message:      "Аутентификация успешна",
		Token:        token,
		GitLabUserID: gitlabUserID,
	}))
}

func handleLogout(c *gin.Context) {
	token, exists := c.Get(AuthToken)
	if !exists {
		c.JSON(http.StatusOK, dto.SuccessResponse(map[string]string{
			"message": "Already logged out",
		}))
		return
	}

	// Удаляем сессию из Redis
	err := deleteUserSession(token.(string))
	if err != nil {
		log.Printf("Error deleting user session: %v", err)
		// Все равно возвращаем успех, так как пользователь все равно выходит
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]string{
		"message": "Logged out successfully",
	}))
}

// Обновим handleMerge для использования данных из контекста
func handleMerge(c *gin.Context) {
	gitlabUserID, exists := c.Get(AuthGitlabUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Не авторизован"))
		return
	}

	var request MergeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "Ошибка парсинга JSON: "+err.Error())
		return
	}

	repository := Repository{AssigneeId: gitlabUserID.(int)}

	if request.Repo == "backend" {
		repository.StandBranch = BackendStandBranch
		repository.ProjectId = BackendProjectID
	} else {
		repository.StandBranch = FrontendStandBranch
		repository.ProjectId = FrontendProjectID
	}

	// Используем userId для отправки сообщений через WebSocket
	err, mrUrl := createLogicMR(request.SourceBranch, repository, gitlabUserID.(int))
	if err != nil {
		c.JSON(200, dto.ErrorResponse(err.Error()))
		return
	}

	resp := dto.SuccessResponse(map[string]string{"mrUrl": mrUrl})
	c.JSON(200, resp)
}
