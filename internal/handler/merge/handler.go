package merge //todo: Перенести в internal/handler/gitlab/merge

import (
	"mergenator/internal/config"
	"mergenator/internal/infr/logger"
	"mergenator/internal/middleware"
	"mergenator/internal/repository/redis"
	"mergenator/internal/service/gitlab"
	"mergenator/internal/service/websocket"
	"mergenator/pkg/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Handler struct {
	cfg         *config.Config
	sessionRepo redis.SessionRepository
	gitlabSvc   gitlab.GitlabService
	log         *zerolog.Logger
}

type mergeRequest struct {
	SourceBranch string `json:"source_branch"`
	Repo         string `json:"repo"`
}

func NewHandler(mergeSvc gitlab.GitlabService, wsService websocket.WebSocketService) *Handler {
	return &Handler{
		gitlabSvc: mergeSvc,
		log:       logger.Get(),
	}
}

func (h *Handler) Merge(c *gin.Context) {
	gitlabUserID, exists := c.Get(middleware.AuthGitlabUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Не авторизован"))
		return
	}

	var request mergeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "Ошибка парсинга JSON: "+err.Error())
		return
	}

	mrUrl, err := h.gitlabSvc.CreateMR(c, request.SourceBranch, request.Repo, gitlabUserID.(int))
	if err != nil {
		c.JSON(200, dto.ErrorResponse(err.Error()))
		h.log.Err(err).Int("user_id", gitlabUserID.(int)).Msg("MR create error")
		return
	}

	resp := dto.SuccessResponse(map[string]string{"mrUrl": mrUrl})
	c.JSON(200, resp)
}
