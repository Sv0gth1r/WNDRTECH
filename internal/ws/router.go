package ws

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

type Router struct {
	log 	*slog.Logger
	hub 	*Hub
	origins map[string]bool
}

func NewRouter(log *slog.Logger, hub *Hub, allowedOrigins []string) *Router {
	set := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		set[o] = true
	}
	return &Router{log: log, hub: hub, origins: set}
}

func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /ws", r.sessionHandler)
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (r *Router) sessionHandler(w http.ResponseWriter, req *http.Request) {
	upgrader := websocket.Upgrader {
		CheckOrigin: func(req *http.Request) bool {
			origin := req.Header.Get("Origin")
			if origin == "" {
				return true // Non-Browser client (CLI, test)
			}
			return r.origins[req.Header.Get("origin")]
		},
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		r.log.Warn("Upgrade failed", "remote", req.RemoteAddr, "err", err)
		return
	}
	s := NewSession(r.hub, r.log, conn, "anonymous") // TODO real auth
	s.Run()
}
