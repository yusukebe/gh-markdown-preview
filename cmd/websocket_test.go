package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWriter(t *testing.T) {
	testFile, err := os.CreateTemp("", "markdown-preview-test")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer os.Remove(testFile.Name())

	_, _ = testFile.Write([]byte("BEFORE.\n"))
	dir := filepath.Dir(testFile.Name())

	watcher, err := createWatcher(dir)

	if err != nil {
		t.Fatalf("%v", err)
	}

	s := httptest.NewServer(http.Handler(wsHandler(watcher, testFile.Name())))

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(50 * time.Millisecond) //XXX

	defer ws.Close()
	defer s.Close()

	_, err = testFile.Write([]byte("AFTER.\n"))
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Flush the write so that Windows reports the change immediately
	// instead of when the file is eventually closed.
	if err := testFile.Sync(); err != nil {
		t.Fatalf("%v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("%v", err)
	}

	actual := string(p)
	expected := "reload"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}
