package middleware

import (
	"net/http"
	"strings"

	"mergenator/internal/service/auth"
	"mergenator/pkg/dto"

	"github.com/gin-gonic/gin"
)

const (
	AuthGitlabUsername = "gitlab_username"
	AuthGitlabUserID   = "gitlab_user_id"
	AuthToken          = "token"
	AuthWSID           = "ws_id"
)

func AuthMiddleware(authSvc auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse("Отсутствует токен авторизации"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Punk" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse("Неверный формат токена"))
			return
		}
		token := parts[1]

		username, userID, err := authSvc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse("Недействительный или истекший токен"))
			return
		}

		wsID, _ := c.Cookie(AuthWSID)

		c.Set(AuthGitlabUsername, username)
		c.Set(AuthGitlabUserID, userID)
		c.Set(AuthToken, token)
		c.Set(AuthWSID, wsID)

		c.Next()
	}
}
