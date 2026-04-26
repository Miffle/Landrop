package presence

import (
	"Test/internal/protocol"
	"encoding/json"
	"log"
)

type DirectMessage struct {
	To     string
	Binary bool
	Data   []byte
}

type WSMessage struct {
	Binary bool
	Data   []byte
}

type Client struct {
	ID   string
	Name string
	Send chan WSMessage
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
		direct:     make(chan DirectMessage, 64),
		remove:     make(chan string),
	}
}

func (h *Hub) Run() {
	for {
		select {

		case id := <-h.remove:
			if c, ok := h.clients[id]; ok {
				log.Printf("[hub] remove dead client %s", id)
				close(c.Send)
				delete(h.clients, id)
			}

		case client := <-h.register:
			log.Printf("[hub] register: %s (%s)", client.Name, client.ID)
			h.clients[client.ID] = client
			h.sendDevices()

		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				log.Printf("[hub] unregister: %s (%s)", client.Name, client.ID)
				close(client.Send)
				delete(h.clients, client.ID)
				h.sendDevices()
			}

		case msg := <-h.direct:
			if client, ok := h.clients[msg.To]; ok {
				wsMsg := WSMessage{Binary: msg.Binary, Data: msg.Data}
				select {
				case client.Send <- wsMsg:
				default:
					log.Printf("[hub] WARN: Send buffer full for client %s (%s), dropping message (len=%d)",
						client.Name, client.ID, len(msg.Data))
				}
			} else {
				log.Printf("[hub] WARN: SendTo unknown client %s", msg.To)
			}

		case message := <-h.broadcast:
			for _, client := range h.clients {
				wsMsg := WSMessage{Binary: false, Data: message}
				select {
				case client.Send <- wsMsg:
				default:
					log.Printf("[hub] WARN: broadcast drop for %s (%s)", client.Name, client.ID)
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

	log.Printf("[hub] broadcast devices: %d client(s)", len(h.clients))

	for _, client := range h.clients {
		wsMsg := WSMessage{Binary: false, Data: data}
		select {
		case client.Send <- wsMsg:
		default:
			log.Printf("[hub] WARN: devices drop for %s (%s)", client.Name, client.ID)
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
	h.direct <- DirectMessage{To: to, Binary: false, Data: data}
}

func (h *Hub) SendBinaryTo(to string, data []byte) {
	h.direct <- DirectMessage{To: to, Binary: true, Data: data}
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}
