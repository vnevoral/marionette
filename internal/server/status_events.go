package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"marionette/internal/config"
)

// statusEventHeartbeat is a variable only so tests can shorten it.
var statusEventHeartbeat = 15 * time.Second

// events streams status transitions as Server-Sent Events (ADR-0008). The
// stream ends when the client goes away, the event source is closed during
// shutdown, or (with access control) at the first heartbeat after the
// client's device was removed (FR-55).
func (api cardAPI) events(w http.ResponseWriter, request *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming is not supported"))
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	subscriber, unsubscribe := api.eventSource.Subscribe()
	defer unsubscribe()
	heartbeat := time.NewTicker(statusEventHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-request.Context().Done():
			return
		case <-api.eventSource.Done():
			return
		case event := <-subscriber:
			payload, err := json.Marshal(struct {
				CardID   string                `json:"cardId"`
				Snapshot config.StatusSnapshot `json:"snapshot"`
			}{CardID: event.CardID, Snapshot: event.Snapshot})
			if err != nil {
				return
			}
			_, _ = fmt.Fprintf(w, "id: %s\nevent: status.changed\ndata: %s\n\n", strconv.FormatUint(event.ID, 10), payload)
			flusher.Flush()
		case <-heartbeat.C:
			if device, ok := currentDevice(request); ok && api.devices != nil && !api.devices.Exists(device.ID) {
				return
			}
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
