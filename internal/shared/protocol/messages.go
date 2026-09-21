package protocol

import "encoding/json"

type Type string

const (
	TypeHello    Type = "hello"
	TypeWelcome  Type = "welcome"
	TypeState    Type = "state"
	TypeSnapshot Type = "snapshot"
	TypeChat     Type = "chat"
	TypePing     Type = "ping"
	TypePong     Type = "pong"
)

type Envelope struct {
	Type Type            `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

func NewEnvelope(t Type, data any) (*Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Envelope{Type: t, Data: raw}, nil
}

func (e *Envelope) Decode(v any) error {
	if len(e.Data) == 0 {
		return nil
	}
	return json.Unmarshal(e.Data, v)
}

type Hello struct {
	Nick    string `json:"nick"`
	Version string `json:"version"`
}

type Welcome struct {
	PlayerID      string `json:"player_id"`
	WorldName     string `json:"world_name"`
	TickRate      int    `json:"tick_rate"`
	ServerVersion string `json:"server_version"`
}

type PlayerState struct {
	ID    string  `json:"id"`
	Nick  string  `json:"nick"`
	X     float32 `json:"x"`
	Y     float32 `json:"y"`
	Z     float32 `json:"z"`
	Yaw   float32 `json:"yaw"`
	Pitch float32 `json:"pitch"`
}

type Snapshot struct {
	Tick    uint64        `json:"tick"`
	Players []PlayerState `json:"players"`
}

type ChatMessage struct {
	From string `json:"from"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

type Ping struct {
	Sent int64 `json:"sent"`
}

type Pong struct {
	Sent     int64 `json:"sent"`
	ServerTS int64 `json:"server_ts"`
}
