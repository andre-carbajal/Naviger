package ws

import (
	"log"
	"net/http"

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
}

func NewHubWithHistorySize(maxHistory int) *Hub {
	if maxHistory < 0 {
		maxHistory = 0
	}
	h := &Hub{
		broadcast:      make(chan []byte),
		Commands:       make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		clients:        make(map[*Client]bool),
		stop:           make(chan bool),
		maxHistory:     maxHistory,
		clearHistory:   make(chan struct{}, 1),
		setHistorySize: make(chan int, 1),
	}
	if maxHistory > 0 {
		h.history = make([][]byte, 0, maxHistory)
	} else {
		h.history = nil
	}
	return h
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

			if h.maxHistory > 0 && len(h.history) > 0 {
				backlog := append([][]byte(nil), h.history...)
				go func(c *Client, b [][]byte) {
					for _, msg := range b {
						select {
						case c.send <- msg:
						default:
							return
						}
					}
				}(client, backlog)
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			if h.maxHistory > 0 {
				h.history = append(h.history, message)
				if len(h.history) > h.maxHistory {
					h.history = h.history[1:]
				}
			}

			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case newSize := <-h.setHistorySize:
			if newSize <= 0 {
				h.maxHistory = 0
				h.history = nil
			} else {
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
			}

		case <-h.clearHistory:
			h.history = nil

		case <-h.stop:
			for client := range h.clients {
				close(client.send)
			}
			h.history = nil
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
	go func() {
		select {
		case h.broadcast <- message:
		default:
		}
	}()
}

func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
