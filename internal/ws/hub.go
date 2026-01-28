package ws

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	Commands   chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan bool

	history        [][]byte
	maxHistory     int
	clearHistory   chan struct{}
	setHistorySize chan int

	snapshotRequests chan *Client

	mu sync.RWMutex
}

func NewHubWithHistorySize(maxHistory int) *Hub {
	if maxHistory < 0 {
		maxHistory = 0
	}
	h := &Hub{
		broadcast:        make(chan []byte, 4096),
		Commands:         make(chan []byte),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		clients:          make(map[*Client]bool),
		stop:             make(chan bool),
		maxHistory:       maxHistory,
		clearHistory:     make(chan struct{}, 1),
		setHistorySize:   make(chan int, 1),
		snapshotRequests: make(chan *Client, 8),
	}
	if maxHistory > 0 {
		h.history = make([][]byte, 0, maxHistory)
	} else {
		h.history = nil
	}
	return h
}

func (h *Hub) GetHistorySnapshot() [][]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.history == nil || len(h.history) == 0 {
		return nil
	}
	copyHist := make([][]byte, len(h.history))
	copy(copyHist, h.history)
	return copyHist
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if client.replay != nil {
					close(client.replay)
				}
			}

		case client := <-h.snapshotRequests:
			h.mu.RLock()
			if h.history != nil && len(h.history) > 0 {
				copyHist := make([][]byte, len(h.history))
				copy(copyHist, h.history)
				h.mu.RUnlock()
				if client.replay == nil {
					client.replay = make(chan []byte, len(copyHist))
				}
				for _, msg := range copyHist {
					select {
					case client.replay <- msg:
					default:
						client.replay <- msg
					}
				}
			} else {
				h.mu.RUnlock()
			}
			h.clients[client] = true

		case message := <-h.broadcast:
			msgCopy := append([]byte(nil), message...)
			if h.maxHistory > 0 {
				h.mu.Lock()
				h.history = append(h.history, msgCopy)
				if len(h.history) > h.maxHistory {
					h.history = h.history[1:]
				}
				h.mu.Unlock()
			}

			for client := range h.clients {
				select {
				case client.send <- msgCopy:
				default:
					close(client.send)
					if client.replay != nil {
						close(client.replay)
					}
					delete(h.clients, client)
				}
			}

		case newSize := <-h.setHistorySize:
			if newSize <= 0 {
				h.maxHistory = 0
				h.mu.Lock()
				h.history = nil
				h.mu.Unlock()
			} else {
				h.mu.Lock()
				if h.history == nil {
					h.history = make([][]byte, 0, newSize)
				} else {
					if len(h.history) > newSize {
						h.history = h.history[len(h.history)-newSize:]
					}
					if cap(h.history) < newSize {
						newHist := make([][]byte, len(h.history), newSize)
						copy(newHist, h.history)
						h.history = newHist
					}
				}
				h.maxHistory = newSize
				h.mu.Unlock()
			}

		case <-h.clearHistory:
			h.mu.Lock()
			h.history = nil
			h.mu.Unlock()

		case <-h.stop:
			for client := range h.clients {
				close(client.send)
			}
			h.mu.Lock()
			h.history = nil
			h.mu.Unlock()
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.stop)
}

func (h *Hub) ClearLogs() {
	select {
	case h.clearHistory <- struct{}{}:
	default:
	}
}

func (h *Hub) SetHistorySize(size int) {
	if size < 0 {
		size = 0
	}
	select {
	case h.setHistorySize <- size:
	default:
		go func() { h.setHistorySize <- size }()
	}
}

func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256), replay: nil}

	go client.writePump()
	go client.readPump()

	select {
	case h.snapshotRequests <- client:
	default:
		go func() { h.snapshotRequests <- client }()
	}
}
