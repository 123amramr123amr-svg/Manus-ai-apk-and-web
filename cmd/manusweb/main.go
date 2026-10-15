// Command manusweb serves the Manus AI web application: a JSON API built on
// the Manus AI Go SDK plus the bundled installable frontend (PWA).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/123amramr123amr-svg/manus-ai-web/internal/app"
	"github.com/123amramr123amr-svg/manus-ai-web/web"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("manus-web ")

	cfg := app.ConfigFromEnv()
	server := app.New(cfg, web.StaticFS(), web.AgentFS())

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s (Manus base URL: %s, server key: %t)",
			cfg.Addr, displayBaseURL(server.Config().BaseURL), server.Config().APIKey != "")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func displayBaseURL(baseURL string) string {
	if baseURL == "" {
		return "https://api.manus.ai"
	}
	return baseURL
}
