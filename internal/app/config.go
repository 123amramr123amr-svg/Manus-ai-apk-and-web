// Package app implements the HTTP server that exposes the Manus AI Go SDK as a
// web application and as a JSON API for the bundled Android WebView client.
package app

import (
	"os"
	"strconv"
	"strings"
)

// Default configuration values.
const (
	DefaultAddr           = ":8080"
	DefaultMaxUploadBytes = 25 << 20 // 25 MiB
	DefaultWebhookEvents  = 200
)

// Config holds every runtime setting of the web application.
type Config struct {
	// Addr is the TCP address the HTTP server listens on (for example ":8080").
	Addr string
	// APIKey is an optional server-side Manus API key. When set, browsers do
	// not have to provide their own key.
	APIKey string
	// BaseURL overrides the Manus API base URL (useful for proxies and tests).
	BaseURL string
	// WebhookToken, when set, must be supplied by Manus webhook calls as the
	// X-Webhook-Token header or the ?token= query parameter.
	WebhookToken string
	// MaxUploadBytes limits the size of a single file upload.
	MaxUploadBytes int64
	// EventBufferSize is how many incoming webhook events are kept in memory.
	EventBufferSize int
	// Version is reported by /api/health and /api/config.
	Version string
}

// ConfigFromEnv builds a Config from environment variables.
//
// Recognised variables: PORT, MANUS_API_KEY, MANUS_API_BASE_URL,
// MANUS_WEBHOOK_TOKEN, MANUS_MAX_UPLOAD_MB, MANUS_EVENT_BUFFER.
func ConfigFromEnv() Config {
	cfg := Config{
		Addr:            DefaultAddr,
		APIKey:          strings.TrimSpace(os.Getenv("MANUS_API_KEY")),
		BaseURL:         strings.TrimSpace(os.Getenv("MANUS_API_BASE_URL")),
		WebhookToken:    strings.TrimSpace(os.Getenv("MANUS_WEBHOOK_TOKEN")),
		MaxUploadBytes:  DefaultMaxUploadBytes,
		EventBufferSize: DefaultWebhookEvents,
		Version:         strings.TrimSpace(os.Getenv("APP_VERSION")),
	}

	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			cfg.Addr = port
		} else {
			cfg.Addr = ":" + port
		}
	}
	if raw := strings.TrimSpace(os.Getenv("MANUS_MAX_UPLOAD_MB")); raw != "" {
		if megabytes, err := strconv.ParseInt(raw, 10, 64); err == nil && megabytes > 0 {
			cfg.MaxUploadBytes = megabytes << 20
		}
	}
	if raw := strings.TrimSpace(os.Getenv("MANUS_EVENT_BUFFER")); raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 {
			cfg.EventBufferSize = size
		}
	}

	return cfg
}

// normalize fills in defaults for zero-valued fields.
func (c Config) normalize() Config {
	if c.Addr == "" {
		c.Addr = DefaultAddr
	}
	if c.MaxUploadBytes <= 0 {
		c.MaxUploadBytes = DefaultMaxUploadBytes
	}
	if c.EventBufferSize <= 0 {
		c.EventBufferSize = DefaultWebhookEvents
	}
	if c.Version == "" {
		c.Version = "dev"
	}
	return c
}
