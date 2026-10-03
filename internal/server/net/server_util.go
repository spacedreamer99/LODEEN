package net

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// broadcast рассылает Envelope всем подключённым клиентам.
func (s *Server) broadcast(t protocol.Type, data any) {
	env, err := protocol.NewEnvelope(t, data)
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		c.enqueue(raw)
	}
}

// readEnvelope читает один frame из r и распаковывает его в Envelope.
func readEnvelope(r io.Reader) (*protocol.Envelope, error) {
	payload, err := protocol.ReadFrame(r)
	if err != nil {
		return nil, err
	}
	var env protocol.Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// newID возвращает 16-символьный hex-идентификатор (8 случайных байт).
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
