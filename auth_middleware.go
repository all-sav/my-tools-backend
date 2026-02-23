package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const (
	AuthGitlabUsername string = "gitlab_username"
	AuthGitlabUserID   string = "gitlab_user_id"
	AuthToken          string = "token"
	AuthWSID           string = "ws_id"
)

// Middleware для проверки токена
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем запросы к логину
		if c.Request.URL.Path == "/auth/login" {
			c.Next()
			return
		}

		// Получаем заголовок Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data": gin.H{
					"message": "Отсутствует токен авторизации.",
				},
			})
			c.Abort()
			return
		}

		// Проверяем формат "Punk <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Punk" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data": gin.H{
					"message": "Неверный формат токена.",
				},
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Проверяем токен в Redis
		username, err := getUsernameByToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data": gin.H{
					"message": "Недействительный или истекший токен",
				},
			})
			c.Abort()
			return
		}

		// Получаем userID из Redis
		userID, err := getGitLabUserID(username)
		if err != nil {
			// Если почему-то нет в Redis, пробуем получить из GitLab
			userID, err = findGitLabUserID(username)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"data": gin.H{
						"message": "Пользователь не найден",
					},
				})
				c.Abort()
				return
			}
		}

		wsId, err := c.Cookie(AuthWSID)
		if err != nil {
			// todo: может тут че то придумаем для обновления websocket - token'a
		}

		// Сохраняем данные пользователя в контексте
		c.Set(AuthGitlabUsername, username)
		c.Set(AuthGitlabUserID, userID)
		c.Set(AuthToken, token)
		c.Set(AuthWSID, wsId)

		c.Next()
	}
}
