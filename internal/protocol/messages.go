package protocol

import "encoding/json"

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type RegisterPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DirectMessagePayload struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

type Device struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DevicesPayload struct {
	Devices []Device `json:"devices"`
}

type FileStart struct {
	To   string `json:"to"`
	From string `json:"from"`
	ID   string `json:"fileId"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type FileChunk struct {
	To   string `json:"to"`
	ID   string `json:"fileId"`
	Data string `json:"data"`
}

type FileEnd struct {
	To string `json:"to"`
	ID string `json:"fileId"`
}

type FileAck struct {
	To     string `json:"to"`
	FileID string `json:"fileId"`
}
