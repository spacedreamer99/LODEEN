package net

import (
	"net"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// acceptLoop принимает новые соединения и запускает для каждого handleConn.
func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.log.Warn("accept error", "err", err)
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// handleConn — оркестратор жизненного цикла одного клиента:
// hello-handshake -> register -> read/write loop -> cleanup.
func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()

	id := newID()
	log := s.log.With("id", id, "remote", conn.RemoteAddr().String())

	client := &Client{
		ID:        id,
		conn:      conn,
		send:      make(chan []byte, sendBufSize),
		done:      make(chan struct{}),
		log:       log,
		inventory: make(map[string]int),
		hunger:    100,
		hp:        100,
	}

	if !s.performHelloHandshake(client) {
		_ = conn.Close()
		return
	}

	s.registerClient(client)

	writerDone := make(chan struct{})
	go func() { defer close(writerDone); s.writeLoop(client) }()

	s.readLoop(client)

	s.unregisterClient(client)
	_ = conn.Close()
	<-writerDone
	log.Info("player left", "nick", client.Nick)
}

// performHelloHandshake читает Hello, валидирует и заполняет client.
// Возвращает false, если handshake не удался (соединение надо закрыть).
func (s *Server) performHelloHandshake(client *Client) bool {
	_ = client.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	env, err := readEnvelope(client.conn)
	if err != nil {
		client.log.Warn("hello read failed", "err", err)
		return false
	}
	if env.Type != protocol.TypeHello {
		client.log.Warn("expected hello", "got", string(env.Type))
		return false
	}
	var hello protocol.Hello
	if err := env.Decode(&hello); err != nil {
		client.log.Warn("hello decode failed", "err", err)
		return false
	}
	if hello.Nick == "" {
		hello.Nick = "anon-" + client.ID[:6]
	}
	client.Nick = hello.Nick
	client.setState(protocol.PlayerState{ID: client.ID, Nick: hello.Nick})
	_ = client.conn.SetReadDeadline(time.Time{})
	return true
}

// registerClient отправляет Welcome+Inventory, добавляет клиента в реестр.
func (s *Server) registerClient(client *Client) {
	client.sendEnvelope(protocol.TypeWelcome, protocol.Welcome{
		PlayerID:      client.ID,
		WorldName:     "lodeen-flat",
		TickRate:      s.tickRate,
		ServerVersion: "dev",
	})
	client.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{
		Items: map[string]int{},
	})

	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()
	s.metrics.PlayersConnected.Inc()
	client.log.Info("player joined", "nick", client.Nick)
}

// unregisterClient удаляет клиента из реестра и освобождает его ракеты.
func (s *Server) unregisterClient(client *Client) {
	client.closeOnce.Do(func() { close(client.done) })
	s.mu.Lock()
	delete(s.clients, client.ID)
	s.mu.Unlock()
	s.metrics.PlayersConnected.Dec()

	s.rockets.Lock()
	for _, r := range s.rockets.Map() {
		if r.OwnerID == client.ID {
			r.Piloted = false
			r.OwnerID = ""
			r.Thrust = 0
		}
	}
	s.rockets.Unlock()
}

// readLoop читает Envelope'ы до ошибки и передаёт их в handleMessage.
func (s *Server) readLoop(c *Client) {
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		env, err := readEnvelope(c.conn)
		if err != nil {
			return
		}
		s.handleMessage(c, env)
	}
}

// writeLoop сливает очередь c.send в сокет.
func (s *Server) writeLoop(c *Client) {
	for {
		select {
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := protocol.WriteFrame(c.conn, payload); err != nil {
				return
			}
		case <-c.done:
			return
		case <-s.done:
			return
		}
	}
}
