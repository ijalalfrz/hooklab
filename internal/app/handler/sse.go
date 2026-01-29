package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/essajiwa/hooklab/internal/app/service"
)

// eventsStreamHandler handles GET /api/stream requests for Server-Sent Events.
// It establishes a persistent connection and streams webhook events in real-time.
// Sends heartbeat pings every 25 seconds to keep the connection alive.
func EventsStreamHandler(a *service.App, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	eventsStreamLoop(a, w, r, flusher, keepAlive.C)
}

// eventsStreamLoop is the main event loop for SSE connections.
// It listens for new events, heartbeat ticks, and context cancellation.
func eventsStreamLoop(a *service.App, w http.ResponseWriter, r *http.Request, flusher http.Flusher, ticks <-chan time.Time) {
	subscriber := a.AddSubscriber()
	defer a.RemoveSubscriber(subscriber)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticks:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case event, ok := <-subscriber:
			if !ok {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(payload)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}