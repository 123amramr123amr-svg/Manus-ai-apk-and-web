package app

import (
	"io"
	"net/http"
	"strings"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// maxWebhookBody is the largest accepted webhook payload.
const maxWebhookBody = 1 << 20 // 1 MiB

// handleListWebhooks returns the webhooks registered in Manus.
func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.ListWebhooks()
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// createWebhookRequest is the body of POST /api/webhooks.
type createWebhookRequest struct {
	URL string `json:"url"`
}

// handleCreateWebhook registers a webhook that points at this server.
func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	var body createWebhookRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "invalid JSON body: " + err.Error()})
		return
	}

	response, err := client.CreateWebhook(&manusai.WebhookConfig{URL: strings.TrimSpace(body.URL)})
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleDeleteWebhook removes a registered webhook.
func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	if err := client.DeleteWebhook(r.PathValue("webhookID")); err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "deleted": true})
}

// handleIncomingWebhook receives Manus notifications. It never calls the Manus
// API itself, so it works without an API key; an optional shared token
// (MANUS_WEBHOOK_TOKEN) can be required.
func (s *Server) handleIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.WebhookToken != "" {
		token := strings.TrimSpace(r.Header.Get(WebhookTokenHeader))
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if token != s.cfg.WebhookToken {
			writeJSON(w, http.StatusUnauthorized, apiError{OK: false, Error: "invalid_webhook_token", Message: "webhook token mismatch"})
			return
		}
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{OK: false, Error: "invalid_body", Message: err.Error()})
		return
	}
	if len(raw) > maxWebhookBody {
		writeJSON(w, http.StatusRequestEntityTooLarge, apiError{OK: false, Error: "payload_too_large", Message: "webhook payload is too large"})
		return
	}

	payload, err := manusai.ParseWebhookPayload(raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{OK: false, Error: "invalid_webhook_payload", Message: err.Error()})
		return
	}

	event := s.events.Add(Event{
		EventType:  payload.EventType,
		TaskDetail: payload.TaskDetail,
		Raw:        string(raw),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"event_id":    event.ID,
		"kind":        event.Kind,
		"completed":   manusai.IsTaskCompleted(payload),
		"needs_input": manusai.IsTaskAskingForInput(payload),
	})
}

// handleListEvents returns the buffered webhook events, newest first.
func (s *Server) handleListEvents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "events": s.events.List()})
}

// handleClearEvents empties the webhook event buffer.
func (s *Server) handleClearEvents(w http.ResponseWriter, _ *http.Request) {
	s.events.Clear()
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "cleared": true})
}
