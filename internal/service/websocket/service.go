package websocket

import (
	"context"
	"net/http"
	"time"
)

type WebSocketService interface {
	HandleConnection(w http.ResponseWriter, r *http.Request)
	SendMessageToUser(ctx context.Context, userID int, message, msgType string) error
}

type WebSocketConfig interface {
	GetTokenTTL() time.Duration
}
