package app

import (
	"net/http"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// handleListMessages returns the conversation of a task.
func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	query := r.URL.Query()
	response, err := client.ListMessages(
		r.PathValue("taskID"),
		intParam(query.Get("limit"), 50),
		query.Get("cursor"),
		query.Get("order"),
		query.Get("verbose") == "true",
	)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// sendMessageRequest is the body of POST /api/tasks/{taskID}/messages.
type sendMessageRequest struct {
	Message     string          `json:"message"`
	Attachments []attachmentRef `json:"attachments"`
}

// handleSendMessage posts a follow-up message to an existing task.
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	var body sendMessageRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "invalid JSON body: " + err.Error()})
		return
	}

	attachments, err := attachmentsFromRefs(body.Attachments)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.SendMessage(r.PathValue("taskID"), body.Message, attachments)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleStopTask asks Manus to stop a running task.
func (s *Server) handleStopTask(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.StopTask(r.PathValue("taskID"))
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// confirmActionRequest is the body of POST /api/tasks/{taskID}/confirm.
type confirmActionRequest struct {
	EventID string                 `json:"event_id"`
	Input   map[string]interface{} `json:"input"`
}

// handleConfirmAction submits the answer for a pending task action.
func (s *Server) handleConfirmAction(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	var body confirmActionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "invalid JSON body: " + err.Error()})
		return
	}

	response, err := client.ConfirmAction(r.PathValue("taskID"), body.EventID, body.Input)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}
