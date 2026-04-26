package server

import (
	"Test/internal/presence"
	"Test/internal/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const writeTimeout = 45 * time.Second

const binaryHeaderSize = 72

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 256,
	WriteBufferSize: 1024 * 256,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func HandleWS(hub *presence.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[ws] new connection from %s", r.RemoteAddr)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws] upgrade error from %s: %v", r.RemoteAddr, err)
			return
		}

		client := &presence.Client{
			Send: make(chan presence.WSMessage, 64),
		}

		go writePump(conn, client)
		readPump(conn, client, hub)
	}
}

func readPump(conn *websocket.Conn, client *presence.Client, hub *presence.Hub) {
	defer func() {
		log.Printf("[ws] readPump done for %s (%s)", client.Name, client.ID)
		hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(binaryHeaderSize + 512*1024)

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseNoStatusReceived,
			) {
				log.Printf("[ws] unexpected close for %s (%s): %v", client.Name, client.ID, err)
			} else {
				log.Printf("[ws] connection closed for %s (%s)", client.Name, client.ID)
			}
			break
		}

		if msgType == websocket.BinaryMessage {
			if len(data) < binaryHeaderSize {
				log.Printf("[ws] binary frame too short (%d bytes) from %s", len(data), client.Name)
				continue
			}
			to := string(data[:36])
			log.Printf("[ws] binary chunk %s → %s (%s)", client.Name, to, formatBytes(int64(len(data)-binaryHeaderSize)))
			hub.SendBinaryTo(to, data)
			continue
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[ws] invalid json from %s: %v", client.ID, err)
			continue
		}

		switch msg.Type {

		case "register":
			var p protocol.RegisterPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Printf("[ws] register error: %v", err)
				continue
			}
			client.ID = p.ID
			client.Name = p.Name
			log.Printf("[ws] registered: %s (%s) from %s", p.Name, p.ID, conn.RemoteAddr())
			hub.Register(client)

		case "file_start":
			var p protocol.FileStart
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Printf("[ws] file_start error: %v", err)
				continue
			}
			p.From = client.ID
			payloadBytes, _ := json.Marshal(p)
			out, _ := json.Marshal(protocol.Message{Type: "file_start", Payload: payloadBytes})
			log.Printf("[ws] file_start: %s → %s | file=%q size=%s",
				client.Name, p.To, p.Name, formatBytes(p.Size))
			hub.SendTo(p.To, out)

		case "file_end":
			var p protocol.FileEnd
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Printf("[ws] file_end error: %v", err)
				continue
			}
			log.Printf("[ws] file_end: %s → %s | fileId=%s", client.Name, p.To, p.ID)
			hub.SendTo(p.To, data)

		case "file_ack":
			var p protocol.FileAck
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Printf("[ws] file_ack error: %v", err)
				continue
			}
			hub.SendTo(p.To, data)

		case "direct_message":
			var p protocol.DirectMessagePayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Printf("[ws] direct_message error: %v", err)
				continue
			}
			payload := map[string]string{"from": client.ID, "text": p.Text}
			payloadBytes, _ := json.Marshal(payload)
			b, _ := json.Marshal(protocol.Message{Type: "direct_message", Payload: payloadBytes})
			hub.SendTo(p.To, b)

		default:
			log.Printf("[ws] unknown message type %q from %s", msg.Type, client.Name)
		}
	}
}

func writePump(conn *websocket.Conn, client *presence.Client) {
	defer conn.Close()

	for msg := range client.Send {
		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			log.Printf("[ws] SetWriteDeadline error for %s: %v", client.Name, err)
			break
		}

		frameType := websocket.TextMessage
		if msg.Binary {
			frameType = websocket.BinaryMessage
		}

		if err := conn.WriteMessage(frameType, msg.Data); err != nil {
			log.Printf("[ws] write error for %s (%s): %v", client.Name, client.ID, err)
			break
		}
	}

	log.Printf("[ws] writePump exited for %s (%s)", client.Name, client.ID)
}

func formatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	}
}
