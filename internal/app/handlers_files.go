package app

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// uploadFileResponse is returned by POST /api/files.
type uploadFileResponse struct {
	OK        bool   `json:"ok"`
	FileID    string `json:"file_id"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	ExpiresAt int64  `json:"expires_at"`
}

// handleUploadFile uploads a multipart file to Manus and returns its file ID,
// which can then be used as an attachment.
func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	limit := s.cfg.MaxUploadBytes
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))
	if err := r.ParseMultipartForm(limit); err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "cannot read upload: " + err.Error()})
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, &manusai.ValidationError{Message: "form field \"file\" is required"})
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		writeAPIError(w, &manusai.Error{Message: "failed to read uploaded file: " + err.Error()})
		return
	}
	if int64(len(content)) > limit {
		writeJSON(w, http.StatusRequestEntityTooLarge, apiError{
			OK:      false,
			Error:   "file_too_large",
			Message: "file exceeds the configured upload limit",
		})
		return
	}

	filename := strings.TrimSpace(header.Filename)
	if filename == "" {
		filename = "upload.bin"
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	created, err := client.CreateFile(filename)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if err := client.UploadFileContent(created.UploadURL, content, contentType); err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, uploadFileResponse{
		OK:        true,
		FileID:    created.FileID,
		Filename:  created.Filename,
		SizeBytes: int64(len(content)),
		ExpiresAt: created.ExpiresAt,
	})
}

// handleGetFile returns the metadata of an uploaded file.
func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	client, err := s.clientFor(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	response, err := client.GetFile(r.PathValue("fileID"))
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}
