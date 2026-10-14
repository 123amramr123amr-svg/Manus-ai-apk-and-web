package app

import (
	"net/http"
	"strings"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// locales accepted by the Manus API.
var supportedLocales = []string{"en-US", "zh-CN", "ar-EG", "fr-FR", "es-ES", "de-DE", "ja-JP", "ko-KR", "pt-BR", "ru-RU"}

// handleHealth reports that the server is alive.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"service": "manus-ai-web",
		"version": s.cfg.Version,
	})
}

// handleConfig tells the client how the server is configured, without ever
// leaking the API key itself.
func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                     true,
		"version":                s.cfg.Version,
		"server_key_configured":  s.cfg.APIKey != "",
		"webhook_token_required": s.cfg.WebhookToken != "",
		"max_upload_bytes":       s.cfg.MaxUploadBytes,
		"base_url":               s.baseURLForDisplay(),
	})
}

// baseURLForDisplay returns the configured Manus base URL or the SDK default.
func (s *Server) baseURLForDisplay() string {
	if s.cfg.BaseURL != "" {
		return s.cfg.BaseURL
	}
	return manusai.DefaultBaseURL
}

// handleProfiles returns the agent profiles and locales offered in the UI.
func (s *Server) handleProfiles(w http.ResponseWriter, _ *http.Request) {
	profiles := make([]map[string]interface{}, 0, len(manusai.AllAgentProfiles()))
	for _, profile := range manusai.AllAgentProfiles() {
		profiles = append(profiles, map[string]interface{}{
			"id":          profile,
			"recommended": !manusai.IsDeprecatedAgentProfile(profile),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":       true,
		"profiles": profiles,
		"locales":  supportedLocales,
	})
}

// apiKeyFromRequest is a small helper for handlers that only need the key.
func (s *Server) apiKeyFromRequest(r *http.Request) string {
	key := strings.TrimSpace(r.Header.Get(APIKeyHeader))
	if key == "" {
		key = s.cfg.APIKey
	}
	return key
}
