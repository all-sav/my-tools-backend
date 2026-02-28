package ws

import (
	"mergenator/internal/service/websocket"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	wsService websocket.WebSocketService
}

func NewHandler(wsService websocket.WebSocketService) *Handler {
	return &Handler{wsService: wsService}
}

func (h *Handler) Handle(c *gin.Context) {
	w := c.Writer
	r := c.Request

	h.wsService.HandleConnection(w, r)
}
