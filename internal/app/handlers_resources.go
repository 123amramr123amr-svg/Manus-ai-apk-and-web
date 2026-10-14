package app

import (
	"net/http"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// handleListUsage returns paginated credit usage records.
func (s *Server) handleListUsage(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	query := r.URL.Query()
	response, err := client.ListUsage(intParam(query.Get("limit"), 50), query.Get("cursor"))
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleAvailableCredits returns the current credit balance.
func (s *Server) handleAvailableCredits(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.GetAvailableCredits()
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListAgents returns the custom agents of the account.
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.ListAgents()
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListProjects returns the projects of the account.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.ListProjects()
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// createProjectRequest is the body of POST /api/projects.
type createProjectRequest struct {
	Name        string `json:"name"`
	Instruction string `json:"instruction"`
}

// handleCreateProject creates a Manus project.
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	var body createProjectRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "invalid JSON body: " + err.Error()})
		return
	}

	response, err := client.CreateProject(body.Name, body.Instruction)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListSkills returns global and project skills.
func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.ListSkills(r.URL.Query().Get("project_id"))
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListConnectors returns the connectors installed in the account.
func (s *Server) handleListConnectors(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.ListConnectors()
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListBrowserClients returns browser clients currently online in Manus.
func (s *Server) handleListBrowserClients(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.ListOnlineBrowserClients()
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}
