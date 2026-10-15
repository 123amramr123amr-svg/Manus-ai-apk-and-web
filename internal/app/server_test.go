package app

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

// recordedRequest captures what the SDK sent to the fake Manus API.
type recordedRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Body    string
	Headers http.Header
}

// fakeManus is an in-process stand-in for https://api.manus.ai.
type fakeManus struct {
	t         *testing.T
	server    *httptest.Server
	mu        sync.Mutex
	requests  []recordedRequest
	statusFor map[string]int // forces an error status for a path
}

func newFakeManus(t *testing.T) *fakeManus {
	t.Helper()

	fake := &fakeManus{t: t, statusFor: map[string]int{}}
	var baseURL string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fake.record(recordedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.Query(),
			Body:    string(body),
			Headers: r.Header.Clone(),
		})

		w.Header().Set("Content-Type", "application/json")
		if status, ok := fake.statusFor[r.URL.Path]; ok {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"ok":false,"error":"forced"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(fake.response(r, baseURL))
	})

	fake.server = httptest.NewServer(handler)
	baseURL = fake.server.URL
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeManus) record(request recordedRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
}

func (f *fakeManus) last() recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		f.t.Fatal("no request reached the fake Manus API")
	}
	return f.requests[len(f.requests)-1]
}

func (f *fakeManus) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := 0
	for _, request := range f.requests {
		if request.Path == path {
			total++
		}
	}
	return total
}

// testStatic is the minimal console frontend used by the HTTP tests.
func testStatic() fstest.MapFS {
	return fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>Manus AI</title>")},
		"app.js":     {Data: []byte("console.log('app');")},
	}
}

// testAgentStatic is the minimal standalone-agent frontend used by the tests.
func testAgentStatic() fstest.MapFS {
	return fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>Manus AI agent</title>")},
		"agent.js":   {Data: []byte("console.log('agent');")},
	}
}

// newTestServer wires the app under test to the fake Manus API.
func newTestServer(t *testing.T, cfg Config) (*Server, *fakeManus) {
	t.Helper()

	fake := newFakeManus(t)
	cfg.BaseURL = fake.server.URL
	if cfg.Version == "" {
		cfg.Version = "test"
	}

	return New(cfg, testStatic(), testAgentStatic()), fake
}

// do performs an HTTP request against the app handler.
func do(t *testing.T, handler http.Handler, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
}

func apiKeyHeaders() map[string]string {
	return map[string]string{APIKeyHeader: "test-key"}
}

// uploadRequest posts a multipart file to /api/files with an API key header.
func uploadRequest(t *testing.T, handler http.Handler, filename, content string) *httptest.ResponseRecorder {
	t.Helper()

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/files", bytes.NewReader(payload.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set(APIKeyHeader, "test-key")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
