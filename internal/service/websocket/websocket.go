package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"mergenator/internal/repository/redis"
)

type Client struct {
	conn   *websocket.Conn
	id     string
	userID int
}

type service struct {
	upgrader    websocket.Upgrader
	clients     map[string]*Client
	mu          sync.Mutex
	sessionRepo redis.SessionRepository
	wsConfig    WebSocketConfig
}

type wsConfig struct {
	ttl time.Duration
}

func (c *wsConfig) GetTokenTTL() time.Duration {
	return c.ttl
}

func NewWebSocketService(allowedOrigins []string, repo redis.SessionRepository, ttl time.Duration) WebSocketService {
	originMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		originMap[o] = true
	}
	return &service{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return originMap[r.Header.Get("Origin")]
			},
		},
		clients:     make(map[string]*Client),
		sessionRepo: repo,
		wsConfig:    &wsConfig{ttl: ttl},
	}
}

func (s *service) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	clientID := uuid.New().String()
	client := &Client{conn: conn, id: clientID, userID: 0}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if client.userID != 0 {
			_ = s.sessionRepo.DeleteWebSocketID(context.Background(), client.userID)
		}
		delete(s.clients, clientID)
		s.mu.Unlock()
	}()

	// Отправляем clientID
	_ = conn.WriteJSON(map[string]string{"clientID": clientID})

	// Установка Pong handler
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	ttl := s.wsConfig.GetTokenTTL()

	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			// Играем в пинг-понг :)
			if msg["type"] == "ping" {
				_ = conn.WriteJSON(map[string]string{"type": "pong"})
				continue
			}
			// Авторизуем юзера(по части websocket-а)
			if msg["type"] == "auth" {
				var authMsg struct {
					Type   string `json:"type"`
					UserID int    `json:"userId"`
				}
				if err := json.Unmarshal(message, &authMsg); err == nil {
					// Сохраняем связку в Redis
					ctx := context.Background()
					if err := s.sessionRepo.StoreWebSocketID(ctx, authMsg.UserID, clientID, ttl); err == nil {
						client.userID = authMsg.UserID
						_ = conn.WriteJSON(map[string]any{
							"type":   "auth_success",
							"userId": authMsg.UserID,
						})
					}
				}
				continue
			}
		}
		log.Printf("Received from %s: %s", clientID, message)
	}
}

func (s *service) SendMessageToUser(ctx context.Context, userID int, message, msgType string) error {
	clientID, err := s.sessionRepo.GetWebSocketID(ctx, userID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	client, ok := s.clients[clientID]
	s.mu.Unlock()
	if !ok || client.conn == nil {
		return nil // клиент не найден или отключён
	}

	msg := struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	}{Message: message, Type: msgType}

	return client.conn.WriteJSON(msg)
}
