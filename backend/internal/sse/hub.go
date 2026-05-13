package sse

import (
	"sync"

	"github.com/google/uuid"
)

// Event is one push to the client.
type Event struct {
	Type string `json:"event"`
	Data any    `json:"data"`
}

// Notifier is the interface the metadata worker depends on.
type Notifier interface {
	Notify(userID uuid.UUID, event Event)
}

// Hub manages per-user SSE channels.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID][]chan Event
}

func NewHub() *Hub {
	return &Hub{clients: make(map[uuid.UUID][]chan Event)}
}

// Subscribe registers a new channel for the user and returns it.
func (h *Hub) Subscribe(userID uuid.UUID) chan Event {
	ch := make(chan Event, 8)
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the registry and closes it.
func (h *Hub) Unsubscribe(userID uuid.UUID, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	chans := h.clients[userID]
	for i, c := range chans {
		if c == ch {
			h.clients[userID] = append(chans[:i], chans[i+1:]...)
			break
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
	close(ch)
}

// Notify implements Notifier — fans the event out to all channels for userID.
func (h *Hub) Notify(userID uuid.UUID, event Event) {
	h.mu.RLock()
	chans := h.clients[userID]
	h.mu.RUnlock()
	for _, ch := range chans {
		select {
		case ch <- event:
		default:
			// channel full — skip rather than block the worker
		}
	}
}
