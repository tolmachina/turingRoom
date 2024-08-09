package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type Client struct {
	conn *websocket.Conn
	role string // "interviewer" or "interviewee"
	send chan []byte
}

type Room struct {
	interviewers [2]*Client
	interviewee  *Client
	broadcast    chan []byte
}

type Message struct {
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

var (
	rooms = make(map[string]*Room)
	mutex sync.Mutex
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for this example
	},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/api/config", handleConfig)

	// Use CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5500"}, // Live Server default port
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
	})
	handler := c.Handler(mux)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}


func handleConfig(w http.ResponseWriter, r *http.Request) {
    config := map[string]string{
        "message": "Config loaded successfully",
    }
    json.NewEncoder(w).Encode(config)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		conn: conn,
		role: r.URL.Query().Get("role"),
		send: make(chan []byte, 256),
	}

	roomID := r.URL.Query().Get("room")
	mutex.Lock()
	room, ok := rooms[roomID]
	if !ok {
		room = &Room{
			broadcast: make(chan []byte, 256),
		}
		rooms[roomID] = room
	}
	mutex.Unlock()

	switch client.role {
	case "interviewer":
		if room.interviewers[0] == nil {
			room.interviewers[0] = client
		} else if room.interviewers[1] == nil {
			room.interviewers[1] = client
		} else {
			log.Println("Room is full for interviewers")
			conn.Close()
			return
		}
	case "interviewee":
		if room.interviewee == nil {
			room.interviewee = client
		} else {
			log.Println("Room already has an interviewee")
			conn.Close()
			return
		}
	default:
		log.Println("Invalid role")
		conn.Close()
		return
	}

	go readPump(client, room)
	go writePump(client)
}

func readPump(client *Client, room *Room) {
	defer func() {
		room.broadcast <- []byte(client.role + " disconnected")
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		msg := Message{
			Sender:    client.role,
			Content:   string(message),
			Timestamp: time.Now().Unix(),
		}
		jsonMsg, _ := json.Marshal(msg)
		room.broadcast <- jsonMsg
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

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func broadcastMessage(room *Room, message []byte) {
	for _, interviewer := range room.interviewers {
		if interviewer != nil {
			select {
			case interviewer.send <- message:
			default:
				close(interviewer.send)
			}
		}
	}
	if room.interviewee != nil {
		select {
		case room.interviewee.send <- message:
		default:
			close(room.interviewee.send)
		}
	}
}

func init() {
	go func() {
		for {
			for _, room := range rooms {
				select {
				case message := <-room.broadcast:
					broadcastMessage(room, message)
				default:
				}
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()
}
