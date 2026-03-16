package auth

import (
	"net/http"

	"mergenator/internal/infr/logger"
	"mergenator/internal/service/auth"
	"mergenator/pkg/dto"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Handler struct {
	authSvc auth.AuthService
	log     *zerolog.Logger
}

func NewHandler(authSvc auth.AuthService) *Handler {
	return &Handler{authSvc: authSvc, log: logger.Get()}
}

func (h *Handler) Login(c *gin.Context) {
	h.log.Debug().Msg("AUTH START")
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		GitLabUser string `json:"gitlab_user"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("Неверный формат запроса"))
		return
	}

	token, userID, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password, req.GitLabUser)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.LoginResponseData{
		Message:      "Аутентификация успешна",
		Token:        token,
		GitLabUserID: userID,
	}))
}

func (h *Handler) Logout(c *gin.Context) {
	tokenVal, exists := c.Get("token") // устанавливается middleware
	if !exists {
		c.JSON(http.StatusOK, dto.SuccessResponse(map[string]string{"message": "Already logged out"}))
		return
	}
	token := tokenVal.(string)
	_ = h.authSvc.Logout(c.Request.Context(), token)
	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]string{"message": "Logged out successfully"}))
}
