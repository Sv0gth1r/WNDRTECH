package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	outBufSize = 64
	maxMsgSize = 64 << 10 // 64KB - set read limit
)

type Session struct {
	conn 	*websocket.Conn
	send 	chan []byte // unique writer, bufferised
	hub 	*Hub
	userID 	string
	log 	*slog.Logger
	mu 		sync.Mutex
	closed bool
}

func NewSession(h *Hub, log *slog.Logger, conn *websocket.Conn, userID string) *Session {
	return &Session {
		conn: 	conn,
		send: 	make(chan []byte, outBufSize),
		hub: 	h,
		userID: userID,
		log: 	log.With("remote", conn.RemoteAddr().String(), "user", userID),
	}
}

func (s *Session) Run() {
	go s.writePump()
	s.readPump()
}

func (s *Session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed { return }
	s.closed = true
	close(s.send)
	_ = s.conn.Close()
	go func() {
		defer func() { _ = recover() }()
		close(s.send)
	}()
}

func (s *Session) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				return
			}
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				s.log.Debug("write failed", "err", err)
				return
			}
		case <-ticker.C:
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Session) readPump() {
	defer s.close()
	s.conn.SetReadLimit(maxMsgSize)
	_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, raw, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		s.handleRaw(raw)
	}
}

func (s *Session) handleRaw(raw []byte) {
	s.log.Info("message received", "bytes", len(raw))

	var generic struct {
		Type string          `json:"type"`
		Body json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &generic); err != nil {
		s.enqueue(json.RawMessage(`{"type":"error","payload":{"msg":"malformed"}}`))
		return
	}

	resp := map[string]any{"type": "echo", "payload": generic.Body}
	s.enqueue(json.RawMessage(mustMarshal(resp)))
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error"}`)
	}
	return b
}

func (s *Session) enqueue(msg []byte) {
	select {
	case s.send <- msg:
	default:
		s.log.Warn("slow client, dropping connection")
		s.close()
	}
}
