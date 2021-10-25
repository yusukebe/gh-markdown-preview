package cmd

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWriter(t *testing.T) {
	testFile, err := ioutil.TempFile(os.TempDir(), "markdown-preview-test")
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, _ = testFile.Write([]byte("BEFORE.\n"))
	s := httptest.NewServer(http.Handler(wsHandler(testFile.Name())))

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	<-time.After(50 * time.Millisecond) //XXX

	defer ws.Close()
	defer s.Close()

	_, err = testFile.Write([]byte("AFTER.\n"))
	if err != nil {
		t.Fatalf("%v", err)
	}

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
