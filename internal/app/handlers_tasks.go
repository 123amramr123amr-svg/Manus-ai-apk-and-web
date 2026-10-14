package app

import (
	"net/http"
	"strconv"
	"strings"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// attachmentRef is the JSON shape accepted for attachments. Exactly one of
// file_id, file_url or file_data should be provided.
type attachmentRef struct {
	FileID   string `json:"file_id"`
	FileURL  string `json:"file_url"`
	Data     string `json:"file_data"`
	MimeType string `json:"mime_type"`
}

func (a attachmentRef) toAttachment() (interface{}, error) {
	switch {
	case strings.TrimSpace(a.FileID) != "":
		return manusai.NewAttachmentFromFileID(a.FileID), nil
	case strings.TrimSpace(a.FileURL) != "":
		return manusai.NewAttachmentFromURL(a.FileURL), nil
	case strings.TrimSpace(a.Data) != "":
		mimeType := a.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		return manusai.NewAttachmentFromBase64(a.Data, mimeType), nil
	default:
		return nil, &manusai.ValidationError{Message: "attachment requires file_id, file_url or file_data"}
	}
}

func attachmentsFromRefs(refs []attachmentRef) ([]interface{}, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	result := make([]interface{}, 0, len(refs))
	for _, ref := range refs {
		attachment, err := ref.toAttachment()
		if err != nil {
			return nil, err
		}
		result = append(result, attachment)
	}
	return result, nil
}

// createTaskRequest is the body of POST /api/tasks.
type createTaskRequest struct {
	Prompt                 string                 `json:"prompt"`
	Title                  string                 `json:"title"`
	AgentProfile           string                 `json:"agent_profile"`
	Locale                 string                 `json:"locale"`
	ProjectID              string                 `json:"project_id"`
	Interactive            *bool                  `json:"interactive"`
	VisibleInTaskList      *bool                  `json:"visible_in_task_list"`
	ShareVisibility        string                 `json:"share_visibility"`
	Connectors             []string               `json:"connectors"`
	EnableSkills           []string               `json:"enable_skills"`
	ForceSkills            []string               `json:"force_skills"`
	Attachments            []attachmentRef        `json:"attachments"`
	StructuredOutputSchema map[string]interface{} `json:"structured_output_schema"`
}

// handleCreateTask starts a new Manus task.
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	var body createTaskRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "invalid JSON body: " + err.Error()})
		return
	}

	attachments, err := attachmentsFromRefs(body.Attachments)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	options := &manusai.TaskOptions{
		AgentProfile:           body.AgentProfile,
		Locale:                 body.Locale,
		Title:                  body.Title,
		ProjectID:              body.ProjectID,
		InteractiveMode:        body.Interactive,
		ShareVisibility:        body.ShareVisibility,
		Connectors:             body.Connectors,
		EnableSkills:           body.EnableSkills,
		ForceSkills:            body.ForceSkills,
		Attachments:            attachments,
		StructuredOutputSchema: body.StructuredOutputSchema,
	}
	if body.VisibleInTaskList != nil {
		hidden := !*body.VisibleInTaskList
		options.HideInTaskList = &hidden
	}

	response, err := client.CreateTask(body.Prompt, options)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleListTasks returns a page of tasks.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	query := r.URL.Query()
	filters := &manusai.TaskFilters{
		Cursor:    query.Get("cursor"),
		Limit:     intParam(query.Get("limit"), 20),
		Order:     query.Get("order"),
		Scope:     query.Get("scope"),
		AgentID:   query.Get("agent_id"),
		ProjectID: query.Get("project_id"),
	}

	response, err := client.GetTasks(filters)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGetTask returns the details of one task.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.GetTask(r.PathValue("taskID"))
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// updateTaskRequest is the body of PATCH /api/tasks/{taskID}.
type updateTaskRequest struct {
	Title             *string `json:"title"`
	ShareVisibility   *string `json:"share_visibility"`
	VisibleInTaskList *bool   `json:"visible_in_task_list"`
}

// handleUpdateTask changes a task title or its visibility.
func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	var body updateTaskRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "invalid JSON body: " + err.Error()})
		return
	}

	response, err := client.UpdateTask(r.PathValue("taskID"), &manusai.TaskUpdate{
		Title:                   body.Title,
		ShareVisibility:         body.ShareVisibility,
		EnableVisibleInTaskList: body.VisibleInTaskList,
	})
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleDeleteTask deletes a task.
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.DeleteTask(r.PathValue("taskID"))
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// intParam parses an integer query parameter, falling back to fallback.
func intParam(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
