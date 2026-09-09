package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"matchserver/internal/ws"
)

var version = "dev" // Injected through -ldflag by Makefile

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	addr := envOr("ADDR", ":8080")
	origins := allowedOrigins()
	hub := ws.NewHub() // TODO add logger
	router := ws.NewRouter(logger, hub, origins)
	
	server := &http.Server{
		Addr:			addr,
		Handler:		router.Handler(), // TODO add logger
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("match server started", "addr", addr, "version", version)
		serverErr <- server.ListenAndServe()
	}()

	select {
		case <-ctx.Done():
			slog.Info("shutdown signal received, draining...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("server shutdown failed", "error", err)
			}
		case err := <-serverErr:
			if err != nil && err != http.ErrServerClosed {
				slog.Error("fatal", "err", err)
				os.Exit(1)
			}
	}
}


func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func allowedOrigins() []string {
	raw := envOr("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
