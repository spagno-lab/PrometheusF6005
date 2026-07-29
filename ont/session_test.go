package ont

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetAndCloseDoesNotWaitForResponseEOF(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-release
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	session := &Session{Client: server.Client(), Endpoint: server.URL}
	done := make(chan struct{})
	go func() {
		session.getAndClose(server.URL)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getAndClose waited for the response body to reach EOF")
	}
}
