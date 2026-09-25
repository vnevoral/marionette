package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestTrackerWaitsForActiveHandler(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	tracker := NewRequestTracker(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
	}))

	done := make(chan struct{})
	go func() {
		tracker.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		close(done)
	}()
	<-started

	waitDone := make(chan struct{})
	go func() {
		tracker.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
		t.Fatal("Wait() returned while a handler was active")
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	<-done
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("Wait() did not return after the handler finished")
	}
}
