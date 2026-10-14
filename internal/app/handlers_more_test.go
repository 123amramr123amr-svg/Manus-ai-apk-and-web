package app

import (
	"net/http"
	"strings"
	"testing"
)

func TestListAndDetailTasks(t *testing.T) {
	server, fake := newTestServer(t, Config{})
	handler := server.Handler()

	list := do(t, handler, http.MethodGet, "/api/tasks?limit=5", "", apiKeyHeaders())
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d (%s)", list.Code, list.Body.String())
	}
	var listBody struct {
		Tasks []struct {
			ID     string  `json:"id"`
			Status string  `json:"status"`
			Credit float64 `json:"credit_usage"`
		} `json:"data"`
	}
	decodeBody(t, list, &listBody)
	if len(listBody.Tasks) != 1 || listBody.Tasks[0].ID != "task_1" || listBody.Tasks[0].Status != "running" {
		t.Fatalf("tasks = %+v", listBody.Tasks)
	}
	if got := fake.last().Query.Get("limit"); got != "5" {
		t.Fatalf("limit query = %q, want 5", got)
	}

	detail := do(t, handler, http.MethodGet, "/api/tasks/task_1", "", apiKeyHeaders())
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d (%s)", detail.Code, detail.Body.String())
	}
	var detailBody struct {
		ID          string `json:"id"`
		AgentStatus string `json:"agent_status"`
	}
	decodeBody(t, detail, &detailBody)
	if detailBody.ID != "task_1" || detailBody.AgentStatus != "finished" {
		t.Fatalf("detail = %+v", detailBody)
	}
	if got := fake.last().Query.Get("task_id"); got != "task_1" {
		t.Fatalf("task_id query = %q", got)
	}
}

func TestUpdateAndDeleteTask(t *testing.T) {
	server, fake := newTestServer(t, Config{})
	handler := server.Handler()

	update := do(t, handler, http.MethodPatch, "/api/tasks/task_1",
		`{"title":"Renamed","visible_in_task_list":true}`, apiKeyHeaders())
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d (%s)", update.Code, update.Body.String())
	}
	var updateBody struct {
		Title string `json:"title"`
	}
	decodeBody(t, update, &updateBody)
	if updateBody.Title != "Renamed" {
		t.Fatalf("updated title = %q", updateBody.Title)
	}
	if !strings.Contains(fake.last().Body, `"enable_visible_in_task_list":true`) {
		t.Fatalf("update payload = %s", fake.last().Body)
	}

	deleted := do(t, handler, http.MethodDelete, "/api/tasks/task_1", "", apiKeyHeaders())
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d (%s)", deleted.Code, deleted.Body.String())
	}
}

func TestMessagesAndSendMessage(t *testing.T) {
	server, fake := newTestServer(t, Config{})
	handler := server.Handler()

	messages := do(t, handler, http.MethodGet, "/api/tasks/task_1/messages?limit=10&verbose=true", "", apiKeyHeaders())
	if messages.Code != http.StatusOK {
		t.Fatalf("messages status = %d (%s)", messages.Code, messages.Body.String())
	}
	var messageBody struct {
		Messages []struct {
			ID        string `json:"id"`
			Type      string `json:"type"`
			Timestamp int64  `json:"timestamp"`
		} `json:"messages"`
	}
	decodeBody(t, messages, &messageBody)
	if len(messageBody.Messages) != 2 {
		t.Fatalf("messages = %+v", messageBody.Messages)
	}
	// RFC3339 timestamps are normalised to Unix milliseconds by the SDK.
	if messageBody.Messages[1].Timestamp != 1704067202000 {
		t.Fatalf("normalised timestamp = %d, want 1704067202000", messageBody.Messages[1].Timestamp)
	}
	if got := fake.last().Query.Get("verbose"); got != "true" {
		t.Fatalf("verbose query = %q", got)
	}

	sent := do(t, handler, http.MethodPost, "/api/tasks/task_1/messages", `{"message":"continue"}`, apiKeyHeaders())
	if sent.Code != http.StatusOK {
		t.Fatalf("send status = %d (%s)", sent.Code, sent.Body.String())
	}
	if !strings.Contains(fake.last().Body, `"text":"continue"`) {
		t.Fatalf("send payload = %s", fake.last().Body)
	}

	stopped := do(t, handler, http.MethodPost, "/api/tasks/task_1/stop", "", apiKeyHeaders())
	if stopped.Code != http.StatusOK {
		t.Fatalf("stop status = %d (%s)", stopped.Code, stopped.Body.String())
	}
}

func TestUploadFile(t *testing.T) {
	server, fake := newTestServer(t, Config{MaxUploadBytes: 1024})

	recorder := uploadRequest(t, server.Handler(), "note.txt", "hello world")
	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status = %d (%s)", recorder.Code, recorder.Body.String())
	}
	var body uploadFileResponse
	decodeBody(t, recorder, &body)
	if body.FileID != "file_1" || body.Filename != "note.txt" || body.SizeBytes != 11 {
		t.Fatalf("upload response = %+v", body)
	}
	if fake.count("/v2/file.upload") != 1 || fake.count("/upload/file_1") != 1 {
		t.Fatalf("expected one create + one PUT, got %d/%d",
			fake.count("/v2/file.upload"), fake.count("/upload/file_1"))
	}
}

func TestUploadFileTooLarge(t *testing.T) {
	server, _ := newTestServer(t, Config{MaxUploadBytes: 4})
	recorder := uploadRequest(t, server.Handler(), "big.bin", "0123456789")

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 (%s)", recorder.Code, recorder.Body.String())
	}
}
