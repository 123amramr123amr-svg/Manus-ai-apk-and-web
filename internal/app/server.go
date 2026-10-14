package app

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// APIKeyHeader carries a per-request Manus API key supplied by the browser or
// the Android client.
const APIKeyHeader = "X-Manus-Api-Key"

// WebhookTokenHeader is the header checked against Config.WebhookToken.
const WebhookTokenHeader = "X-Webhook-Token"

// Server exposes the Manus SDK over HTTP: a JSON API under /api and the
// bundled single-page application everywhere else.
type Server struct {
	cfg     Config
	static  fs.FS
	events  *EventStore
	client  func(apiKey string) (*manusai.Client, error)
	handler http.Handler
}

// New creates a Server serving staticFS as the frontend.
func New(cfg Config, staticFS fs.FS) *Server {
	cfg = cfg.normalize()

	s := &Server{
		cfg:    cfg,
		static: staticFS,
		events: NewEventStore(cfg.EventBufferSize),
	}
	s.client = func(apiKey string) (*manusai.Client, error) {
		options := []manusai.ClientOption{manusai.WithTimeout(60 * time.Second)}
		if cfg.BaseURL != "" {
			options = append(options, manusai.WithBaseURL(cfg.BaseURL))
		}
		return manusai.NewClient(apiKey, options...)
	}
	s.handler = s.routes()

	return s
}

// Handler returns the fully configured HTTP handler.
func (s *Server) Handler() http.Handler { return s.handler }

// Config returns the normalized configuration.
func (s *Server) Config() Config { return s.cfg }

// routes registers every endpoint. Go 1.22 method-and-pattern routing is used,
// so "GET /api/tasks/{taskID}" only matches GET requests.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/profiles", s.handleProfiles)

	mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("GET /api/tasks/{taskID}", s.handleGetTask)
	mux.HandleFunc("PATCH /api/tasks/{taskID}", s.handleUpdateTask)
	mux.HandleFunc("DELETE /api/tasks/{taskID}", s.handleDeleteTask)
	mux.HandleFunc("GET /api/tasks/{taskID}/messages", s.handleListMessages)
	mux.HandleFunc("POST /api/tasks/{taskID}/messages", s.handleSendMessage)
	mux.HandleFunc("POST /api/tasks/{taskID}/stop", s.handleStopTask)
	mux.HandleFunc("POST /api/tasks/{taskID}/confirm", s.handleConfirmAction)

	mux.HandleFunc("POST /api/files", s.handleUploadFile)
	mux.HandleFunc("GET /api/files/{fileID}", s.handleGetFile)

	mux.HandleFunc("GET /api/usage", s.handleListUsage)
	mux.HandleFunc("GET /api/credits", s.handleAvailableCredits)
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	mux.HandleFunc("GET /api/skills", s.handleListSkills)
	mux.HandleFunc("GET /api/connectors", s.handleListConnectors)
	mux.HandleFunc("GET /api/browser-clients", s.handleListBrowserClients)

	mux.HandleFunc("GET /api/webhooks", s.handleListWebhooks)
	mux.HandleFunc("POST /api/webhooks", s.handleCreateWebhook)
	mux.HandleFunc("DELETE /api/webhooks/{webhookID}", s.handleDeleteWebhook)
	mux.HandleFunc("GET /api/webhook/events", s.handleListEvents)
	mux.HandleFunc("DELETE /api/webhook/events", s.handleClearEvents)
	mux.HandleFunc("POST /api/webhook", s.handleIncomingWebhook)

	mux.HandleFunc("/", s.handleStatic)

	return s.withMiddleware(mux)
}

// withMiddleware adds security headers and request logging.
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; "+
					"connect-src 'self'; manifest-src 'self'; base-uri 'none'; form-action 'self'")
		}

		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

// clientFor builds a Manus client using the request API key when present and
// the server-side key otherwise.
func (s *Server) clientFor(r *http.Request) (*manusai.Client, error) {
	apiKey := s.apiKeyFromRequest(r)
	if apiKey == "" {
		return nil, errMissingAPIKey
	}
	return s.client(apiKey)
}

// handleStatic serves the embedded single-page application, falling back to
// index.html so client-side routes keep working on a hard refresh. Unknown
// /api/ paths (including method mismatches on known paths) are reported as 404
// because this catch-all pattern handles every remaining method.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusNotFound, apiError{OK: false, Error: "not_found", Message: "unknown API endpoint"})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.static == nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}

	file, err := s.static.Open(name)
	if err != nil {
		s.serveIndex(w, r)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		s.serveIndex(w, r)
		return
	}

	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		s.serveIndex(w, r)
		return
	}

	if name == "sw.js" || name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=604800")
	}
	http.ServeContent(w, r, name, info.ModTime(), seeker)
}

// serveIndex writes index.html, the SPA entry point.
func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	if s.static == nil {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(s.static, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
}
