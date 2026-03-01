package webhook //todo: Перенести в internal/handler/gitlab/webhook

import (
	"fmt"
	"net/http"
	"strings"

	"mergenator/internal/infr/logger"
	"mergenator/internal/service/gitlab"
	"mergenator/pkg/dto"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type GitLabWebhook struct {
	ObjectKind string `json:"object_kind"`
	Ref        string `json:"ref"`
	Project    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
}

type Handler struct {
	gitlabService gitlab.GitlabService
	webhookToken  string
	log           *zerolog.Logger
}

func NewHandler(service gitlab.GitlabService, webhookToken string) *Handler {
	return &Handler{
		gitlabService: service,
		webhookToken:  webhookToken,
		log:           logger.Get(),
	}
}

func (h *Handler) Handle(c *gin.Context) {
	// Валидация токена
	token := c.GetHeader("X-Gitlab-Token")
	if token != h.webhookToken {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Invalid token"))
		h.log.Err(fmt.Errorf("invalid webhook token")).Str("token", token)
		return
	}

	var webhookData GitLabWebhook
	if err := c.ShouldBindJSON(&webhookData); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("Invalid JSON: "+err.Error()))
		return
	}

	// Нас интересуют только push-события
	if webhookData.ObjectKind != "push" {
		c.Status(http.StatusOK)
		return
	}

	// Извлекаем название ветки из ref (refs/heads/branch-name)
	branch := strings.TrimPrefix(webhookData.Ref, "refs/heads/")
	projectID := webhookData.Project.ID

	// Вызываем сервис для обработки push
	if err := h.gitlabService.HandlePush(c.Request.Context(), branch, string(projectID)); err != nil {
		// Логируем ошибку, но клиенту возвращаем 200, чтобы GitLab не паниковал
		h.log.Err(err).Msg("Webhook error")
	}

	c.Status(http.StatusOK)
}
