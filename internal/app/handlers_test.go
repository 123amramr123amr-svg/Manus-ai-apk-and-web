package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestHealthAndConfig(t *testing.T) {
	server, _ := newTestServer(t, Config{})
	handler := server.Handler()

	health := do(t, handler, http.MethodGet, "/api/health", "", nil)
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}
	var healthBody map[string]interface{}
	decodeBody(t, health, &healthBody)
	if healthBody["service"] != "manus-ai-web" {
		t.Fatalf("health service = %v", healthBody["service"])
	}

	config := do(t, handler, http.MethodGet, "/api/config", "", nil)
	var configBody map[string]interface{}
	decodeBody(t, config, &configBody)
	if configBody["server_key_configured"] != false {
		t.Fatalf("server_key_configured = %v, want false", configBody["server_key_configured"])
	}
	if configBody["base_url"] == "" {
		t.Fatal("base_url should be reported")
	}
}

func TestProfilesEndpoint(t *testing.T) {
	server, _ := newTestServer(t, Config{})
	recorder := do(t, server.Handler(), http.MethodGet, "/api/profiles", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
		Locales []string `json:"locales"`
	}
	decodeBody(t, recorder, &body)
	if len(body.Profiles) == 0 || len(body.Locales) == 0 {
		t.Fatalf("profiles/locales empty: %+v", body)
	}
}

func TestMissingAPIKeyReturnsUnauthorized(t *testing.T) {
	server, fake := newTestServer(t, Config{})
	recorder := do(t, server.Handler(), http.MethodGet, "/api/tasks", "", nil)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	var body apiError
	decodeBody(t, recorder, &body)
	if body.Error != "missing_api_key" {
		t.Fatalf("error = %q, want missing_api_key", body.Error)
	}
	if fake.count("/v2/task.list") != 0 {
		t.Fatal("the SDK must not be called without a key")
	}
}

func TestServerKeyIsUsedWhenClientOmitsIt(t *testing.T) {
	server, fake := newTestServer(t, Config{APIKey: "server-key"})
	recorder := do(t, server.Handler(), http.MethodGet, "/api/tasks", "", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", recorder.Code, recorder.Body.String())
	}
	if got := fake.last().Headers.Get("x-manus-api-key"); got != "server-key" {
		t.Fatalf("api key header = %q, want server-key", got)
	}
}

func TestClientKeyOverridesServerKey(t *testing.T) {
	server, fake := newTestServer(t, Config{APIKey: "server-key"})
	do(t, server.Handler(), http.MethodGet, "/api/tasks", "", apiKeyHeaders())

	if got := fake.last().Headers.Get("x-manus-api-key"); got != "test-key" {
		t.Fatalf("api key header = %q, want test-key", got)
	}
}

func TestCreateTaskBuildsV2Payload(t *testing.T) {
	server, fake := newTestServer(t, Config{})
	body := `{
		"prompt": "summarise the docs",
		"title": "Docs",
		"agent_profile": "manus-1.6",
		"locale": "ar-EG",
		"interactive": true,
		"visible_in_task_list": false,
		"attachments": [{"file_id": "file_1"}, {"file_url": "https://example.com/a.pdf"}]
	}`

	recorder := do(t, server.Handler(), http.MethodPost, "/api/tasks", body, apiKeyHeaders())
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	decodeBody(t, recorder, &response)
	if response["task_id"] != "task_1" {
		t.Fatalf("task_id = %v", response["task_id"])
	}

	var sent struct {
		Message struct {
			Content []map[string]interface{} `json:"content"`
		} `json:"message"`
		AgentProfile    string `json:"agent_profile"`
		Locale          string `json:"locale"`
		InteractiveMode *bool  `json:"interactive_mode"`
		HideInTaskList  *bool  `json:"hide_in_task_list"`
	}
	if err := json.Unmarshal([]byte(fake.last().Body), &sent); err != nil {
		t.Fatalf("decode request body %q: %v", fake.last().Body, err)
	}
	if len(sent.Message.Content) != 3 {
		t.Fatalf("content parts = %d, want text + 2 attachments", len(sent.Message.Content))
	}
	if sent.Message.Content[0]["type"] != "text" || sent.Message.Content[0]["text"] != "summarise the docs" {
		t.Fatalf("first content part = %v", sent.Message.Content[0])
	}
	if sent.Message.Content[1]["file_id"] != "file_1" {
		t.Fatalf("attachment 1 = %v", sent.Message.Content[1])
	}
	if sent.Message.Content[2]["file_url"] != "https://example.com/a.pdf" {
		t.Fatalf("attachment 2 = %v", sent.Message.Content[2])
	}
	if sent.AgentProfile != "manus-1.6" || sent.Locale != "ar-EG" {
		t.Fatalf("profile/locale = %q/%q", sent.AgentProfile, sent.Locale)
	}
	if sent.InteractiveMode == nil || !*sent.InteractiveMode {
		t.Fatal("interactive_mode should be true")
	}
	if sent.HideInTaskList == nil || !*sent.HideInTaskList {
		t.Fatal("hide_in_task_list should be true when visible_in_task_list is false")
	}
}

func TestCreateTaskRejectsUnknownField(t *testing.T) {
	server, _ := newTestServer(t, Config{})
	recorder := do(t, server.Handler(), http.MethodPost, "/api/tasks", `{"prompt":"x","bogus":1}`, apiKeyHeaders())
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestCreateTaskPropagatesManusError(t *testing.T) {
	server, fake := newTestServer(t, Config{})
	fake.statusFor["/v2/task.create"] = http.StatusUnauthorized

	recorder := do(t, server.Handler(), http.MethodPost, "/api/tasks", `{"prompt":"x"}`, apiKeyHeaders())
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (%s)", recorder.Code, recorder.Body.String())
	}
	var body apiError
	decodeBody(t, recorder, &body)
	if body.Error != "manus_auth_error" {
		t.Fatalf("error = %q, want manus_auth_error", body.Error)
	}
	if !strings.Contains(body.Message, "forced") {
		t.Fatalf("message should include the upstream body: %q", body.Message)
	}
}
