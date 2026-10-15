package app

import (
	"net/http"
	"strings"
	"testing"
)

func TestWebhookFlow(t *testing.T) {
	server, _ := newTestServer(t, Config{WebhookToken: "secret"})
	handler := server.Handler()

	payload := `{"event_type":"task_stopped","task_detail":{"task_id":"task_1","stop_reason":"finish","title":"Done"}}`

	unauthorized := do(t, handler, http.MethodPost, "/api/webhook", payload, nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("status without token = %d, want 401", unauthorized.Code)
	}

	accepted := do(t, handler, http.MethodPost, "/api/webhook", payload,
		map[string]string{WebhookTokenHeader: "secret"})
	if accepted.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", accepted.Code, accepted.Body.String())
	}
	var acceptedBody struct {
		Kind      string `json:"kind"`
		Completed bool   `json:"completed"`
	}
	decodeBody(t, accepted, &acceptedBody)
	if acceptedBody.Kind != "completed" || !acceptedBody.Completed {
		t.Fatalf("webhook classification = %+v", acceptedBody)
	}

	invalid := do(t, handler, http.MethodPost, "/api/webhook", `{"nope":true}`,
		map[string]string{WebhookTokenHeader: "secret"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid payload status = %d, want 400", invalid.Code)
	}

	var eventsBody struct {
		Events []Event `json:"events"`
	}
	events := do(t, handler, http.MethodGet, "/api/webhook/events", "", nil)
	decodeBody(t, events, &eventsBody)
	if len(eventsBody.Events) != 1 || eventsBody.Events[0].Kind != "completed" {
		t.Fatalf("events = %+v", eventsBody.Events)
	}

	cleared := do(t, handler, http.MethodDelete, "/api/webhook/events", "", nil)
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear status = %d", cleared.Code)
	}
	after := do(t, handler, http.MethodGet, "/api/webhook/events", "", nil)
	decodeBody(t, after, &eventsBody)
	if len(eventsBody.Events) != 0 {
		t.Fatalf("events after clear = %+v", eventsBody.Events)
	}
}

func TestResourcesEndpoints(t *testing.T) {
	server, _ := newTestServer(t, Config{})
	handler := server.Handler()

	paths := []string{
		"/api/credits",
		"/api/usage?limit=3",
		"/api/agents",
		"/api/projects",
		"/api/skills",
		"/api/connectors",
		"/api/browser-clients",
		"/api/webhooks",
		"/api/files/file_1",
	}
	for _, path := range paths {
		recorder := do(t, handler, http.MethodGet, path, "", apiKeyHeaders())
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d (%s)", path, recorder.Code, recorder.Body.String())
		}
	}

	created := do(t, handler, http.MethodPost, "/api/projects", `{"name":"New"}`, apiKeyHeaders())
	if created.Code != http.StatusOK {
		t.Fatalf("create project status = %d (%s)", created.Code, created.Body.String())
	}

	webhook := do(t, handler, http.MethodPost, "/api/webhooks", `{"url":"https://example.com/hook"}`, apiKeyHeaders())
	if webhook.Code != http.StatusOK {
		t.Fatalf("create webhook status = %d (%s)", webhook.Code, webhook.Body.String())
	}

	removed := do(t, handler, http.MethodDelete, "/api/webhooks/wh_2", "", apiKeyHeaders())
	if removed.Code != http.StatusOK {
		t.Fatalf("delete webhook status = %d (%s)", removed.Code, removed.Body.String())
	}
}

func TestAgentAppIsServed(t *testing.T) {
	server, _ := newTestServer(t, Config{})
	handler := server.Handler()

	redirect := do(t, handler, http.MethodGet, "/agent", "", nil)
	if redirect.Code != http.StatusTemporaryRedirect {
		t.Fatalf("redirect status = %d, want 307", redirect.Code)
	}

	index := do(t, handler, http.MethodGet, "/agent/", "", nil)
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "Manus AI agent") {
		t.Fatalf("agent index = %d %q", index.Code, index.Body.String())
	}
	if got := index.Header().Get("Content-Security-Policy"); !strings.Contains(got, "connect-src *") {
		t.Fatalf("agent CSP should allow cross-origin API calls, got %q", got)
	}

	script := do(t, handler, http.MethodGet, "/agent/agent.js", "", nil)
	if script.Code != http.StatusOK || !strings.Contains(script.Body.String(), "console.log") {
		t.Fatalf("agent script = %d %q", script.Code, script.Body.String())
	}

	fallback := do(t, handler, http.MethodGet, "/agent/deep/link", "", nil)
	if fallback.Code != http.StatusOK || !strings.Contains(fallback.Body.String(), "Manus AI agent") {
		t.Fatalf("agent fallback = %d", fallback.Code)
	}
}

func TestStaticFilesAndSPAFallback(t *testing.T) {
	server, _ := newTestServer(t, Config{})
	handler := server.Handler()

	index := do(t, handler, http.MethodGet, "/", "", nil)
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "Manus AI") {
		t.Fatalf("index = %d %q", index.Code, index.Body.String())
	}
	if index.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("CSP header missing")
	}

	asset := do(t, handler, http.MethodGet, "/app.js", "", nil)
	if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), "console.log") {
		t.Fatalf("asset = %d %q", asset.Code, asset.Body.String())
	}
	if got := asset.Header().Get("Cache-Control"); !strings.Contains(got, "max-age") {
		t.Fatalf("asset cache-control = %q", got)
	}

	spa := do(t, handler, http.MethodGet, "/tasks/deep/link", "", nil)
	if spa.Code != http.StatusOK || !strings.Contains(spa.Body.String(), "Manus AI") {
		t.Fatalf("SPA fallback = %d %q", spa.Code, spa.Body.String())
	}

	unknownAPI := do(t, handler, http.MethodGet, "/api/nope", "", nil)
	if unknownAPI.Code != http.StatusNotFound {
		t.Fatalf("unknown api status = %d, want 404", unknownAPI.Code)
	}
	if !strings.Contains(unknownAPI.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unknown api content-type = %q", unknownAPI.Header().Get("Content-Type"))
	}

	wrongMethod := do(t, handler, http.MethodPost, "/api/health", "", nil)
	// The catch-all pattern serves unknown /api/ paths, so a method mismatch on
	// a known path is reported as 404 rather than 405.
	if wrongMethod.Code != http.StatusNotFound {
		t.Fatalf("wrong method status = %d, want 404", wrongMethod.Code)
	}
}

func TestEventStoreRingBuffer(t *testing.T) {
	store := NewEventStore(2)
	store.Add(Event{EventType: "task_created"})
	store.Add(Event{EventType: "task_stopped", TaskDetail: map[string]interface{}{"stop_reason": "ask"}})
	store.Add(Event{EventType: "task_created"})

	events := store.List()
	if len(events) != 2 {
		t.Fatalf("buffered events = %d, want 2", len(events))
	}
	if events[0].Kind != "created" || events[1].Kind != "waiting_for_input" {
		t.Fatalf("kinds = %q, %q", events[0].Kind, events[1].Kind)
	}
	if events[0].ID == events[1].ID {
		t.Fatal("event IDs must be unique")
	}

	store.Clear()
	if len(store.List()) != 0 {
		t.Fatal("Clear should empty the buffer")
	}
}
