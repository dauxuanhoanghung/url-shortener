package sse

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHub_Subscribe_ReceivesEvent(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	want := Event{Type: "url_deleted", Data: map[string]any{"url_id": "abc"}}
	hub.Notify(userID, want)

	select {
	case got := <-ch:
		if got.Type != want.Type {
			t.Errorf("event type: got %q want %q", got.Type, want.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestHub_Notify_WrongUser_NotReceived(t *testing.T) {
	hub := NewHub()
	userA := uuid.New()
	userB := uuid.New()

	ch := hub.Subscribe(userA)
	defer hub.Unsubscribe(userA, ch)

	hub.Notify(userB, Event{Type: "url_deleted"})

	select {
	case e := <-ch:
		t.Errorf("user A should not receive event for user B, got %v", e)
	case <-time.After(50 * time.Millisecond):
		// correct — nothing arrived
	}
}

func TestHub_MultipleSubscribers_BothReceive(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	ch1 := hub.Subscribe(userID)
	ch2 := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch1)
	defer hub.Unsubscribe(userID, ch2)

	hub.Notify(userID, Event{Type: "url_deleted"})

	for _, ch := range []chan Event{ch1, ch2} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event on subscriber")
		}
	}
}

func TestHub_Unsubscribe_StopsDelivery(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	ch := hub.Subscribe(userID)
	hub.Unsubscribe(userID, ch) // immediately unsubscribe

	// Notify after unsubscribe — must not panic (closed channel).
	// The hub must not send to a closed channel; it removes it first.
	hub.Notify(userID, Event{Type: "url_deleted"})

	// Verify the internal registry is cleaned up.
	hub.mu.RLock()
	remaining := len(hub.clients[userID])
	hub.mu.RUnlock()
	if remaining != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", remaining)
	}
}

func TestHub_Unsubscribe_OneOfMany(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	ch1 := hub.Subscribe(userID)
	ch2 := hub.Subscribe(userID)
	hub.Unsubscribe(userID, ch1)

	hub.mu.RLock()
	remaining := len(hub.clients[userID])
	hub.mu.RUnlock()
	if remaining != 1 {
		t.Errorf("expected 1 subscriber after removing one of two, got %d", remaining)
	}

	// ch2 must still work.
	hub.Notify(userID, Event{Type: "ping"})
	select {
	case <-ch2:
	case <-time.After(time.Second):
		t.Fatal("remaining subscriber did not receive event")
	}
	hub.Unsubscribe(userID, ch2)
}

func TestHub_FullChannel_DoesNotBlock(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	// Fill the channel buffer (cap=8) without draining.
	for i := 0; i < 20; i++ {
		hub.Notify(userID, Event{Type: "url_deleted"})
	}
	// If we get here, Notify never blocked — success.
}

func TestHub_ConcurrentNotify_Safe(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	ch := hub.Subscribe(userID)
	defer hub.Unsubscribe(userID, ch)

	// Drain in background so channel doesn't fill.
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
			case <-stop:
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Notify(userID, Event{Type: "url_deleted"})
		}()
	}
	wg.Wait()
	close(stop)
}
