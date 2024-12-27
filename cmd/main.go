package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/zhekagigs/turing-room/llm"
	"github.com/zhekagigs/turing-room/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Role string

const (
	Interviewer Role = "interviewer"
	Polee       Role = "polee"
	AI          Role = "ai"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	role Role
	room *Room
}

type Message struct {
	Content string `json:"content"`
	From    Role   `json:"from"`
	To      Role   `json:"to"`
}

type Room struct {
	ID          string
	Interviewer *Client
	Polee       *Client
	AI          llm.AIClient
	broadcast   chan Message
	messages    *MessageStorage
}

type MessageStorage struct {
}

var rooms = make(map[string]*Room)
var roomsMutex = sync.Mutex{}

func init() {
	go func() {
		for {
			roomsMutex.Lock()
			for _, room := range rooms {
				select {
				case message := <-room.broadcast:
					go broadcastMessage(room, message)
				default:
				}
			}
			roomsMutex.Unlock()
			time.Sleep(time.Millisecond * 100)
		}
	}()
}

func main() {
	logger.Initialize("logs/app.log")
	logger.Info("Starting application...")

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/api/config", handleConfig)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5500", "http://localhost:8000"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
	})
	handler := c.Handler(mux)

	logger.Info("Server starting on :8080")
	logger.Fatal(http.ListenAndServe(":8080", handler))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection:", err)
		return
	}

	roomID := r.URL.Query().Get("room")
	role := Role(r.URL.Query().Get("role"))

	if role != Interviewer && role != Polee {
		logger.Error("Invalid role:", role)
		conn.Close()
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256), role: role}

	roomsMutex.Lock()
	room, ok := rooms[roomID]
	if !ok {
		room = &Room{
			ID:        roomID,
			broadcast: make(chan Message),
			AI:        llm.NewOllamaClient("http://localhost:11434/v1"),
		}
		rooms[roomID] = room
		go handleRoomMessages(room)
	}

	switch role {
	case Interviewer:
		if room.Interviewer != nil {
			roomsMutex.Unlock()
			logger.Error("Room already has an interviewer")
			conn.Close()
			return
		}
		room.Interviewer = client
	case Polee:
		if room.Polee != nil {
			roomsMutex.Unlock()
			logger.Error("Room already has a polee")
			conn.Close()
			return
		}
		room.Polee = client
	}
	roomsMutex.Unlock()

	client.room = room

	go readPump(client)
	go writePump(client)

	logger.Info("New WebSocket connection established for role:", role)
}

func handleRoomMessages(room *Room) {
	for {
		msg := <-room.broadcast
		switch msg.To {
		case Interviewer:
			sendToClient(room.Interviewer, msg)
		case Polee:
			sendToClient(room.Polee, msg)
		case AI:
			aiResponse, err := room.AI.GenerateResponse(msg.Content, "Pretend you are human")
			if err != nil {
				logger.Error("Error generating AI response:", err)
				continue
			}
			aiMsg := Message{Content: aiResponse, From: AI, To: Interviewer}
			sendToClient(room.Interviewer, aiMsg)
		}
	}
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling config request")
	config := map[string]string{
		"message": "Config loaded successfully",
	}
	json.NewEncoder(w).Encode(config)
}

func readPump(client *Client) {
	defer func() {
		client.conn.Close()
		if client.room != nil {
			if client.role == Interviewer {
				client.room.Interviewer = nil
			} else if client.role == Polee {
				client.room.Polee = nil
			}
		}
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("Error reading message:", err)
			}
			break
		}
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Error("Error unmarshaling message:", err)
			continue
		}
		msg.From = client.role
		client.room.broadcast <- msg
	}
}

func writePump(client *Client) {
	defer client.conn.Close()
	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("Error writing message:", err)
				return
			}
		}
	}
}

func sendToClient(client *Client, msg Message) {
	if client == nil {
		return
	}
	messageJSON, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Error marshaling message:", err)
		return
	}
	client.send <- messageJSON
}

func broadcastMessage(room *Room, message Message) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		logger.Error("Error marshaling message:", err)
		return
	}

	sendToClient := func(client *Client) {
		if client == nil {
			return
		}
		select {
		case client.send <- messageJSON:
		default:
			close(client.send)
			if client.role == Interviewer {
				room.Interviewer = nil
			} else if client.role == Polee {
				room.Polee = nil
			}
		}
	}

	switch message.To {
	case Interviewer:
		sendToClient(room.Interviewer)
	case Polee:
		sendToClient(room.Polee)
	case AI:
		aiResponse, err := room.AI.GenerateResponse(message.Content, "Pretend you are human")
		if err != nil {
			logger.Error("Error generating AI response:", err)
			return
		}
		aiMsg := Message{Content: aiResponse, From: AI, To: Interviewer}
		fmt.Println("aiMsg", aiMsg)
		sendToClient(room.Interviewer)
	default:
		sendToClient(room.Interviewer)
		sendToClient(room.Polee)
	}
}
