package presence

import (
	"Test/internal/protocol"
	"encoding/json"
)

type DirectMessage struct {
	To   string
	Data []byte
}

type Client struct {
	ID   string
	Name string
	Send chan []byte
}

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	direct     chan DirectMessage
	remove     chan string
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		direct:     make(chan DirectMessage, 16),
		remove:     make(chan string),
	}
}

func (h *Hub) Run() {
	for {
		select {

		case id := <-h.remove:
			if c, ok := h.clients[id]; ok {
				close(c.Send)
				delete(h.clients, id)
			}

		case client := <-h.register:
			h.clients[client.ID] = client
			h.sendDevices()

		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				close(client.Send)
				delete(h.clients, client.ID)
				h.sendDevices()
			}

		case msg := <-h.direct:
			if client, ok := h.clients[msg.To]; ok {
				select {
				case client.Send <- msg.Data:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}

		case message := <-h.broadcast:
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
		}
	}
}

func (h *Hub) sendDevices() {
	var devices []protocol.Device
	for _, c := range h.clients {
		devices = append(devices, protocol.Device{
			ID:   c.ID,
			Name: c.Name,
		})
	}

	payloadBytes, _ := json.Marshal(protocol.DevicesPayload{Devices: devices})
	data, _ := json.Marshal(protocol.Message{
		Type:    "devices",
		Payload: payloadBytes,
	})

	for _, client := range h.clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.clients, client.ID)
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) SendTo(to string, data []byte) {
	h.direct <- DirectMessage{To: to, Data: data}
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}
