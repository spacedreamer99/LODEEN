package ws

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Proxy struct {
	Log      *slog.Logger
	GameAddr string
}

func (p *Proxy) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			p.Log.Warn("ws upgrade failed", "err", err)
			return
		}
		defer wsConn.Close()

		tcpConn, err := net.Dial("tcp", p.GameAddr)
		if err != nil {
			p.Log.Warn("tcp dial failed", "err", err, "addr", p.GameAddr)
			_ = wsConn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					websocket.CloseInternalServerErr, "game server unavailable"),
			)
			return
		}
		defer tcpConn.Close()

		done := make(chan struct{}, 1)

		go func() {
			defer func() { done <- struct{}{} }()
			for {
				payload, err := protocol.ReadFrame(tcpConn)
				if err != nil {
					return
				}
				if err := wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return
				}
			}
		}()

		for {
			select {
			case <-done:
				return
			default:
			}

			_, payload, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			if err := protocol.WriteFrame(tcpConn, payload); err != nil {
				return
			}
		}
	}
}
