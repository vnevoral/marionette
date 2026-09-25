package server

import (
	"net/http"
	"sync"
)

// RequestTracker tracks active HTTP handlers so shutdown can wait for them.
type RequestTracker struct {
	handler http.Handler

	mu     sync.Mutex
	active int
	idle   chan struct{}
}

// NewRequestTracker wraps a handler with active-request tracking.
func NewRequestTracker(handler http.Handler) *RequestTracker {
	tracker := &RequestTracker{handler: handler, idle: make(chan struct{})}
	close(tracker.idle)
	return tracker
}

// ServeHTTP delegates a request and tracks it until the handler returns.
func (tracker *RequestTracker) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	tracker.mu.Lock()
	if tracker.active == 0 {
		tracker.idle = make(chan struct{})
	}
	tracker.active++
	tracker.mu.Unlock()
	defer func() {
		tracker.mu.Lock()
		tracker.active--
		if tracker.active == 0 {
			close(tracker.idle)
		}
		tracker.mu.Unlock()
	}()
	tracker.handler.ServeHTTP(w, request)
}

// Wait blocks until all currently active handlers have returned.
func (tracker *RequestTracker) Wait() {
	tracker.mu.Lock()
	idle := tracker.idle
	tracker.mu.Unlock()
	<-idle
}
