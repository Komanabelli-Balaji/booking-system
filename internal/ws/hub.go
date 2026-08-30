package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	movieRooms map[string]map[*Client]bool
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		movieRooms: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Broadcast(movieID string) {
	h.mu.RLock()
	clients, ok := h.movieRooms[movieID]

	if !ok {
		h.mu.RUnlock()
		return
	}

	clientsCopy := make([]*Client, 0, len(clients))
	for client := range clients {
		clientsCopy = append(clientsCopy, client)
	}
	h.mu.RUnlock()

	msg, _ := json.Marshal(map[string]string{
		"type":     "seats_changed",
		"movie_id": movieID,
	})

	for _, client := range clientsCopy {
		select {
		case client.send <- msg:
		default:
		}
	}
}

func (h *Hub) subscribe(client *Client, movieID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.movieID != "" {
		if room, ok := h.movieRooms[client.movieID]; ok {
			delete(room, client)
			if len(room) == 0 {
				delete(h.movieRooms, client.movieID)
			}
		}
	}

	client.movieID = movieID

	if movieID == "" {
		return
	}

	if h.movieRooms[movieID] == nil {
		h.movieRooms[movieID] = make(map[*Client]bool)
	}
	h.movieRooms[movieID][client] = true
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.movieID != "" {
		if room, ok := h.movieRooms[client.movieID]; ok {
			delete(room, client)
			if len(room) == 0 {
				delete(h.movieRooms, client.movieID)
			}
		}
	}
	close(client.send)
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256)}

	go client.writePump()
	go client.readPump()
}
