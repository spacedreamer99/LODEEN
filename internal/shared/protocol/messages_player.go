package protocol

// --- Сессия ---

type Hello struct {
	Nick     string `json:"nick"`
	Version  string `json:"version"`
	ColorHex string `json:"color_hex,omitempty"` // "#RRGGBB", пусто — сервер выберет
}

type Welcome struct {
	PlayerID      string `json:"player_id"`
	WorldName     string `json:"world_name"`
	TickRate      int    `json:"tick_rate"`
	ServerVersion string `json:"server_version"`
}

// --- Состояние игрока ---

type PlayerState struct {
	ID       string  `json:"id"`
	Nick     string  `json:"nick"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Z        float32 `json:"z"`
	Yaw      float32 `json:"yaw"`
	Pitch    float32 `json:"pitch"`
	Hunger   float32 `json:"hunger"`
	HP       int     `json:"hp"`
	RTTms    int32   `json:"rtt_ms,omitempty"`
	ColorHex string  `json:"color_hex,omitempty"`
}

// Teleport — сервер приказывает клиенту переместиться в точку.
type Teleport struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// --- Чат ---

type ChatMessage struct {
	From string `json:"from"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

// --- Ping/Pong ---

type Ping struct {
	Sent int64 `json:"sent"`
}

type Pong struct {
	Sent     int64 `json:"sent"`
	ServerTS int64 `json:"server_ts"`
}

// --- Инвентарь и предметы ---

type SelectItem struct {
	Item string `json:"item"`
}

type EatFruit struct{}

type InventoryUpdate struct {
	Items map[string]int `json:"items"`
}

type PickupItem struct {
	ResourceID string `json:"resource_id"`
}
