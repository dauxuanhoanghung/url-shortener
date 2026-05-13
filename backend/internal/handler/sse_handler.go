package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dauxuanhoanghung/url-shortener/internal/sse"
	"github.com/gin-gonic/gin"
)

type SSEHandler struct {
	hub *sse.Hub
}

func NewSSEHandler(hub *sse.Hub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

// Stream handles GET /events — long-lived SSE connection per authenticated user.
func (h *SSEHandler) Stream(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.hub.Subscribe(userID)
	defer h.hub.Unsubscribe(userID, ch)

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Type, data)
			if f, ok := c.Writer.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
