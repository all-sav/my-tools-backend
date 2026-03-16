package settingshandler

import (
	"fmt"
	"mergenator/internal/infr/logger"
	"mergenator/internal/service/settings"
	"mergenator/pkg/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Handler struct {
	settingsSvc settings.SettingsService
	log         *zerolog.Logger
}

type saveSettingsRequest struct {
	Module string `json:"module"`
	Data   any    `json:"data"`
}

func NewHandler(settingsSvc settings.SettingsService) *Handler {
	return &Handler{
		settingsSvc: settingsSvc,
		log:         logger.Get(),
	}
}

func (h *Handler) Get(c *gin.Context) {
	module := c.Query("module")
	if h.settingsSvc.HasModule(module) == false {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(fmt.Sprintf("Module %s not available", module)))
		return
	}

	settings, err := h.settingsSvc.Get(module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Server error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(settings))
}

func (h *Handler) Save(c *gin.Context) {
	var data saveSettingsRequest
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("Invalid data"))
		return
	}

	if h.settingsSvc.HasModule(data.Module) == false {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(fmt.Sprintf("Module %s not available", data.Module)))
		return
	}

	if err := h.settingsSvc.Update(data.Module, data.Data); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Server error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]string{"message": "Settings saved"}))
}
