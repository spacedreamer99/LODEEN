package net_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	servernet "github.com/spacedreamer99/lodeen/internal/server/net"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func TestHandshakeAndChat(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := &servernet.Metrics{
		PlayersConnected: prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_players"}),
		TicksTotal:       prometheus.NewCounter(prometheus.CounterOpts{Name: "test_ticks"}),
		ChatMessages:     prometheus.NewCounter(prometheus.CounterOpts{Name: "test_chat"}),
	}

	srv := servernet.New("127.0.0.1:0", 20, log, m)
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	}()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := writeEnv(conn, protocol.TypeHello, protocol.Hello{Nick: "tester"}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	env := readEnv(t, conn)
	if env.Type != protocol.TypeWelcome {
		t.Fatalf("expected welcome, got %s", env.Type)
	}

	if err := writeEnv(conn, protocol.TypeChat, protocol.ChatMessage{Text: "hi"}); err != nil {
		t.Fatalf("write chat: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		env := readEnv(t, conn)
		if env.Type == protocol.TypeChat {
			var cm protocol.ChatMessage
			if err := env.Decode(&cm); err != nil {
				t.Fatalf("decode chat: %v", err)
			}
			if cm.From != "tester" || cm.Text != "hi" {
				t.Fatalf("unexpected chat: %+v", cm)
			}
			return
		}
	}
	t.Fatal("chat not received")
}

func writeEnv(w io.Writer, t protocol.Type, data any) error {
	env, err := protocol.NewEnvelope(t, data)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return protocol.WriteFrame(w, raw)
}

func readEnv(t *testing.T, r io.Reader) *protocol.Envelope {
	t.Helper()
	_ = r.(net.Conn).SetReadDeadline(time.Now().Add(2 * time.Second))
	payload, err := protocol.ReadFrame(r)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	var env protocol.Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &env
}
