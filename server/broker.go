package main

import (
	"encoding/json"
	"sync"
)

// BrokerPublisher is the interface for broadcasting events to SSE clients.
type BrokerPublisher interface {
	BroadcastJSON(eventType string, data interface{})
}

// SSEEvent represents a Server-Sent Event to broadcast to connected clients.
type SSEEvent struct {
	Type string // "tracking_started", "tracking_stopped", "state"
	Data []byte // JSON payload
}

// Broker manages SSE client subscriptions and broadcasts events.
type Broker struct {
	mu      sync.Mutex
	clients map[chan SSEEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		clients: make(map[chan SSEEvent]struct{}),
	}
}

// Subscribe returns a channel that receives SSE events and an unsubscribe function.
func (b *Broker) Subscribe() (ch chan SSEEvent, unsubscribe func()) {
	ch = make(chan SSEEvent, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe = func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
	}
	return ch, unsubscribe
}

// Broadcast sends an event to all connected clients. Non-blocking: slow clients
// that have a full buffer will have the event dropped.
func (b *Broker) Broadcast(event SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// ClientCount returns the number of connected clients.
func (b *Broker) ClientCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.clients)
}

// BroadcastJSON is a convenience method that marshals data to JSON and broadcasts.
func (b *Broker) BroadcastJSON(eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	b.Broadcast(SSEEvent{Type: eventType, Data: jsonData})
}
