package server

import (
	"Test/internal/presence"
	"Test/internal/protocol"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func HandleWS(hub *presence.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}

		client := &presence.Client{
			Send: make(chan []byte, 8),
		}

		go writePump(conn, client)
		readPump(conn, client, hub)
	}
}

func readPump(conn *websocket.Conn, client *presence.Client, hub *presence.Hub) {
	defer func() {
		hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(768 * 1024)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Println("read error:", err)
			}
			break
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Println("invalid json:", err)
			continue
		}

		switch msg.Type {

		case "register":
			var p protocol.RegisterPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Println("register payload error:", err)
				continue
			}
			client.ID = p.ID
			client.Name = p.Name
			hub.Register(client)

		case "file_start":
			var p protocol.FileStart
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Println("file_start payload error:", err)
				continue
			}
			hub.SendTo(p.To, data)

		case "file_chunk":
			var p protocol.FileChunk
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Println("file_chunk payload error:", err)
				continue
			}
			hub.SendTo(p.To, data)

		case "file_end":
			var p protocol.FileEnd
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Println("file_end payload error:", err)
				continue
			}
			hub.SendTo(p.To, data)

		case "direct_message":
			var p protocol.DirectMessagePayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Println("direct_message payload error:", err)
				continue
			}

			payload := map[string]string{
				"from": client.ID,
				"text": p.Text,
			}
			payloadBytes, _ := json.Marshal(payload)
			b, _ := json.Marshal(protocol.Message{
				Type:    "direct_message",
				Payload: payloadBytes,
			})
			hub.SendTo(p.To, b)

		default:
			log.Println("unknown message type:", msg.Type)
		}
	}
}

func writePump(conn *websocket.Conn, client *presence.Client) {
	defer conn.Close()

	for msg := range client.Send {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
