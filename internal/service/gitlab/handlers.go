package gitlab

import (
	"mergenator/internal/service/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleMerge(c *gin.Context) {
	gitlabUserID, exists := c.Get(auth.AuthGitlabUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Не авторизован"))
		return
	}

	var request MergeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "Ошибка парсинга JSON: "+err.Error())
		return
	}

	repository := services.Repository{AssigneeId: gitlabUserID.(int)}

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
