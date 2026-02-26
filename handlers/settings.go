package handlers

import (
	"net/http"
	"time"

	"mergenator/db"

	"github.com/gin-gonic/gin"
)

type SaveSettingsRequest struct {
	BackendProjectID    string `json:"backend_project_id"`
	FrontendProjectID   string `json:"frontend_project_id"`
	BackendStandBranch  string `json:"backend_stand_branch"`
	FrontendStandBranch string `json:"frontend_stand_branch"`
	CIMainBranch        string `json:"ci_main_branch"`
	RequiredPrefix      string `json:"required_prefix"`
	Prefix              string `json:"prefix"`
	CIPrefix            string `json:"ci_prefix"`
}

type SettingsResponse struct {
	Success bool                   `json:"success"`
	Data    *db.MergenatorSettings `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// GET /api/settings - получить настройки текущего пользователя
func GetSettings(c *gin.Context) {
	settings, err := db.GetMergenatorSettings(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, SettingsResponse{
			Success: false,
			Error:   "Ошибка получения настроек: " + err.Error(),
		})
		return
	}

	if settings == nil {
		// Возвращаем пустой объект, если настроек нет
		settings = &db.MergenatorSettings{GitLabUserID: userID.(int)}
	}

	c.JSON(http.StatusOK, SettingsResponse{
		Success: true,
		Data:    settings,
	})
}

// POST /api/settings - сохранить настройки
func SaveSettings(c *gin.Context) {
	userID, exists := c.Get(main.AuthGitlabUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, SettingsResponse{
			Success: false,
			Error:   "Не авторизован",
		})
		return
	}

	var req SaveSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SettingsResponse{
			Success: false,
			Error:   "Неверный формат запроса: " + err.Error(),
		})
		return
	}

	settings := db.MergenatorSettings{
		GitLabUserID:        userID.(int),
		BackendProjectID:    req.BackendProjectID,
		FrontendProjectID:   req.FrontendProjectID,
		BackendStandBranch:  req.BackendStandBranch,
		FrontendStandBranch: req.FrontendStandBranch,
		CIMainBranch:        req.CIMainBranch,
		RequiredPrefix:      req.RequiredPrefix,
		Prefix:              req.Prefix,
		CIPrefix:            req.CIPrefix,
		UpdatedAt:           time.Now().Unix(),
	}

	if err := db.SaveMergenatorSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, SettingsResponse{
			Success: false,
			Error:   "Ошибка сохранения настроек: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SettingsResponse{
		Success: true,
		Data:    &settings,
	})
}

// DELETE /api/settings - сбросить настройки
func ResetSettings(c *gin.Context) {
	userID, exists := c.Get(main.AuthGitlabUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, SettingsResponse{
			Success: false,
			Error:   "Не авторизован",
		})
		return
	}

	if err := db.DeleteUserSettings(userID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, SettingsResponse{
			Success: false,
			Error:   "Ошибка сброса настроек: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SettingsResponse{
		Success: true,
	})
}
