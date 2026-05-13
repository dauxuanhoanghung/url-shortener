package handler

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var testSSEUserID = uuid.New()

// flushRecorder wraps httptest.ResponseRecorder and implements http.Flusher
// by writing each flush to a pipe, so tests can read lines in real time.
type flushRecorder struct {
	*httptest.ResponseRecorder
	pw *io.PipeWriter
}

func (f *flushRecorder) Write(b []byte) (int, error) {
	f.ResponseRecorder.Write(b)
	return f.pw.Write(b)
}

func (f *flushRecorder) Flush() {
	f.ResponseRecorder.Flush()
}

// collectSSELines runs the SSE handler for at most dur and returns every
// non-blank line received. The handler exits when ctx is cancelled.
func collectSSELines(t *testing.T, hub *sse.Hub, userID uuid.UUID, dur time.Duration) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	pr, pw := io.Pipe()
	rec := &flushRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		pw:               pw,
	}

	r := gin.New()
	h := NewSSEHandler(hub)
	r.GET("/events", func(c *gin.Context) {
		c.Set("userID", userID.String())
		h.Stream(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	go func() {
		r.ServeHTTP(rec, req)
		pw.Close()
	}()

	var lines []string
	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestSSEHandler_ResponseHeaders(t *testing.T) {
	hub := sse.NewHub()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := gin.New()
	h := NewSSEHandler(hub)
	r.GET("/events", func(c *gin.Context) {
		c.Set("userID", testSSEUserID.String())
		h.Stream(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	r.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: got %q want text/event-stream", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control: got %q want no-cache", cc)
	}
}

func TestSSEHandler_DeliversUrlDeletedEvent(t *testing.T) {
	hub := sse.NewHub()
	userID := uuid.New()

	linesCh := make(chan []string, 1)
	go func() {
		linesCh <- collectSSELines(t, hub, userID, 500*time.Millisecond)
	}()

	// Give the handler time to subscribe before notifying.
	time.Sleep(30 * time.Millisecond)

	hub.Notify(userID, sse.Event{
		Type: "url_deleted",
		Data: map[string]any{"url_id": "abc123", "short_code": "abc123"},
	})

	// Wait briefly then cancel by letting the timeout fire.
	// collectSSELines will return once the pipe closes (context cancelled).
	select {
	case lines := <-linesCh:
		var foundEvent, foundData bool
		for _, line := range lines {
			if line == "event: url_deleted" {
				foundEvent = true
			}
			if strings.HasPrefix(line, "data:") && strings.Contains(line, "abc123") {
				foundData = true
			}
		}
		if !foundEvent {
			t.Errorf("missing 'event: url_deleted' line; got: %v", lines)
		}
		if !foundData {
			t.Errorf("missing data line with url_id; got: %v", lines)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE lines")
	}
}

func TestSSEHandler_DoesNotDeliverToWrongUser(t *testing.T) {
	hub := sse.NewHub()
	userID := uuid.New()

	linesCh := make(chan []string, 1)
	go func() {
		// Short timeout — we expect nothing to arrive.
		linesCh <- collectSSELines(t, hub, userID, 200*time.Millisecond)
	}()

	time.Sleep(30 * time.Millisecond)

	// Notify a completely different user.
	hub.Notify(uuid.New(), sse.Event{Type: "url_deleted", Data: map[string]any{"url_id": "xyz"}})

	lines := <-linesCh
	for _, line := range lines {
		if strings.Contains(line, "url_deleted") {
			t.Errorf("should not receive event for a different user; got: %q", line)
		}
	}
}
