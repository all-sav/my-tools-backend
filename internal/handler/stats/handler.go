package stats

import (
	"net/http"

	dockersvc "mergenator/internal/service/docker"
	"mergenator/internal/service/gitlab"
	"mergenator/pkg/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	gitlabSvc gitlab.GitlabService
	dockerSvc dockersvc.DockerService
}

func NewHandler(gitlabSvc gitlab.GitlabService, dockerSvc dockersvc.DockerService) *Handler {
	return &Handler{
		gitlabSvc: gitlabSvc,
		dockerSvc: dockerSvc,
	}
}

func (h *Handler) Get(c *gin.Context) {
	stats, err := h.gitlabSvc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Failed to load stats"))
		return
	}

	dockerStats, err := h.dockerSvc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Failed to load docker stats"))
		return
	}

	stats.Docker = dockerStats
	c.JSON(http.StatusOK, dto.SuccessResponse(stats))
}
