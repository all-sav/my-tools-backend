package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	WSMessageTypeDefault = "default"
	WSMessageTypeError   = "error"
	WSMessageTypeSuccess = "success"
	WSMessageTypeHeader  = "header"
)

type WSMessage struct {
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}

type AuthMessage struct {
	Type     string `json:"type"`
	UserId   int    `json:"userId"`
	ClientId string `json:"clientId"`
}

type Client struct {
	conn   *websocket.Conn
	id     string
	userId int // Добавляем userId для связи с авторизованным пользователем
}

var (
	allowedOrigins = map[string]bool{
		"https://localhost:3000":             true,
		"https://bh-a1.cow-and-dog.com:8075": true,
	}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return allowedOrigins[origin]
		},
	}
	wsClients = make(map[string]*Client)
	mutex     = sync.Mutex{}
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка установки WebSocket:", err)
		return
	}
	defer conn.Close()

	conn.SetReadLimit(65536)
	conn.SetPongHandler(func(appData string) error {
		log.Println("Получен pong от клиента")
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	clientID := uuid.New().String()
	client := &Client{
		conn:   conn,
		id:     clientID,
		userId: 0, // По умолчанию не авторизован
	}

	mutex.Lock()
	wsClients[clientID] = client
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		// Если клиент был авторизован, удаляем его из Redis
		if client.userId != 0 {
			redisClient.Del(ctx, rKeyGitLabUserIDToWebsocketID+strconv.Itoa(client.userId))
		}
		delete(wsClients, clientID)
		mutex.Unlock()
	}()

	// Отправляем ID клиенту
	err = conn.WriteJSON(map[string]string{"clientID": clientID})
	if err != nil {
		log.Printf("Ошибка отправки ID клиенту: %v", err)
		return
	}

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket ошибка: %v", err)
			}
			break
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Пробуем распарсить как JSON
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			// Обрабатываем ping
			if msg["type"] == "ping" {
				pongMsg := map[string]string{"type": "pong"}
				if err := conn.WriteJSON(pongMsg); err != nil {
					log.Printf("Ошибка отправки pong: %v", err)
				}
				continue
			}

			// Обрабатываем auth сообщение
			if msg["type"] == "auth" {
				var authMsg AuthMessage
				if err := json.Unmarshal(message, &authMsg); err == nil {
					// Сохраняем связку userId -> websocketId в Redis
					ttl, _ := time.ParseDuration(os.Getenv("TOKEN_TTL"))
					if ttl == 0 {
						ttl = 24 * time.Hour
					}

					err = redisClient.Set(ctx, rKeyGitLabUserIDToWebsocketID+strconv.Itoa(authMsg.UserId), clientID, ttl).Err()
					if err != nil {
						log.Printf("Ошибка сохранения websocket ID в Redis: %v", err)
					} else {
						// Обновляем userId у клиента
						client.userId = authMsg.UserId

						// Отправляем подтверждение
						conn.WriteJSON(map[string]interface{}{
							"type":   "auth_success",
							"userId": authMsg.UserId,
						})
						log.Printf("WebSocket авторизован для пользователя %d с ID %s", authMsg.UserId, clientID)
					}
				}
				continue
			}
		}

		log.Printf("Получено от %s: %s", clientID, message)
	}
}

// Обновленная функция отправки сообщений - проверяет актуальность соединения
func sendMessageByID(userId int, message string, WSMessageType string) {
	// Ищем clientId по userId в Redis
	clientId, err := redisClient.Get(ctx, rKeyGitLabUserIDToWebsocketID+strconv.Itoa(userId)).Result()
	if err != nil {
		log.Printf("WebSocket ID для пользователя %d не найден в Redis", userId)
		return
	}

	mutex.Lock()
	client, exists := wsClients[clientId]
	mutex.Unlock()

	if !exists || client.conn == nil {
		log.Printf("WebSocket клиент %s не найден в памяти", clientId)
		return
	}

	wsMessage := WSMessage{
		Message: message,
		Type:    WSMessageType,
	}

	jsonData, err := json.Marshal(wsMessage)
	if err != nil {
		log.Println("Ошибка сериализации:", err)
		return
	}

	err = client.conn.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		log.Printf("Ошибка отправки сообщения по ws: %v", err)
		// При ошибке удаляем клиента
		mutex.Lock()
		delete(wsClients, clientId)
		mutex.Unlock()
		redisClient.Del(ctx, rKeyGitLabUserIDToWebsocketID+strconv.Itoa(userId))
	}
}
