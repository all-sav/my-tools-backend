package stats

import (
	"net/http"

	"mergenator/internal/service/gitlab"
	"mergenator/pkg/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	gitlabSvc gitlab.GitlabService
}

func NewHandler(gitlabSvc gitlab.GitlabService) *Handler {
	return &Handler{gitlabSvc: gitlabSvc}
}

func (h *Handler) Get(c *gin.Context) {
	stats, err := h.gitlabSvc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Failed to load stats"))
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(stats))
}
