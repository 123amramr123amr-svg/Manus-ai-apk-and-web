package app

import (
	"strconv"
	"sync"
	"time"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// Event is a webhook notification received from Manus, kept in memory so the
// UI can display a live activity feed.
type Event struct {
	ID         string                 `json:"id"`
	Kind       string                 `json:"kind"`
	EventType  string                 `json:"event_type"`
	ReceivedAt time.Time              `json:"received_at"`
	TaskDetail map[string]interface{} `json:"task_detail,omitempty"`
	Raw        string                 `json:"raw,omitempty"`
}

// EventStore is a bounded, concurrency-safe ring buffer of webhook events.
type EventStore struct {
	mu       sync.RWMutex
	capacity int
	nextID   int64
	events   []Event
}

// NewEventStore creates an EventStore that keeps at most capacity events.
func NewEventStore(capacity int) *EventStore {
	if capacity <= 0 {
		capacity = DefaultWebhookEvents
	}
	return &EventStore{capacity: capacity}
}

// Add appends an event, dropping the oldest one when the buffer is full.
func (s *EventStore) Add(event Event) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	event.ID = strconv.FormatInt(s.nextID, 10)
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now().UTC()
	}
	if event.Kind == "" {
		event.Kind = classifyEvent(event.EventType, event.TaskDetail)
	}

	s.events = append(s.events, event)
	if len(s.events) > s.capacity {
		s.events = s.events[len(s.events)-s.capacity:]
	}

	return event
}

// List returns the buffered events, newest first.
func (s *EventStore) List() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Event, 0, len(s.events))
	for i := len(s.events) - 1; i >= 0; i-- {
		result = append(result, s.events[i])
	}
	return result
}

// Clear removes every buffered event.
func (s *EventStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

// classifyEvent turns a webhook payload into a coarse UI category.
func classifyEvent(eventType string, detail map[string]interface{}) string {
	payload := &manusai.WebhookPayload{EventType: eventType, TaskDetail: detail}

	switch {
	case manusai.IsTaskCompleted(payload):
		return "completed"
	case manusai.IsTaskAskingForInput(payload):
		return "waiting_for_input"
	case manusai.IsTaskStopped(payload):
		return "stopped"
	case manusai.IsTaskCreated(payload):
		return "created"
	default:
		return "other"
	}
}
