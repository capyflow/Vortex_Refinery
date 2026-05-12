package adapter

import (
	"encoding/json"
	"io"
	"net/http"
)

// Handler handles webhook requests
type Handler struct {
	eventBus interface {
		PushEvent(ctx interface{}, event interface{}) error
	}
}

// WebhookRequest represents a webhook request payload
type WebhookRequest struct {
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
}

// WebhookResponse represents a webhook response
type WebhookResponse struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

// parseRequest parses the incoming webhook request
func (h *Handler) parseRequest(r *http.Request) (*WebhookRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	return &req, nil
}

// sendResponse sends a JSON response
func sendResponse(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func sendError(w http.ResponseWriter, status int, message string) {
	sendResponse(w, status, map[string]string{"error": message})
}
