package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// startWS serves wsHandler the way Serve does, for a fresh directory that
// contains README.md, and returns the handler, the file and the WebSocket URL.
func startWS(t *testing.T) (http.Handler, string, string) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "README.md")
	if err := os.WriteFile(file, []byte("v0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	watcher, err := createWatcher(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := wsHandler(watcher, file)
	s := httptest.NewServer(h)
	t.Cleanup(func() {
		s.Close()
		watcher.Close()
	})
	return h, file, "ws" + strings.TrimPrefix(s.URL, "http")
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// readReload fails the test unless c receives "reload" within a few seconds.
func readReload(t *testing.T, c *websocket.Conn) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, p, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("no reload notification: %v", err)
	}
	if string(p) != "reload" {
		t.Fatalf("got %q, want %q", p, "reload")
	}
}

// waitReload saves file every 200ms until c receives "reload". Saving more
// than once keeps the test independent of the watcher's 100ms lock and of
// slow file events on Windows.
func waitReload(t *testing.T, c *websocket.Conn, file string) {
	t.Helper()
	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for i := 0; ; i++ {
			_ = os.WriteFile(file, []byte(fmt.Sprintf("save %d\n", i)), 0o644)
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}()
	defer func() {
		close(stop)
		<-stopped
	}()
	readReload(t, c)
}

func TestReloadBroadcastsToAllClients(t *testing.T) {
	_, file, url := startWS(t)
	first := dialWS(t, url)
	second := dialWS(t, url)

	waitReload(t, first, file)
	readReload(t, second)
}

func TestReloadSurvivesClosedClient(t *testing.T) {
	_, file, url := startWS(t)
	closed := dialWS(t, url)
	waitReload(t, closed, file)

	// The user closes the tab: the browser sends a close frame and drops the
	// connection. The file keeps changing while no tab is open.
	_ = closed.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseGoingAway, ""), time.Now().Add(time.Second))
	_ = closed.Close()
	for i := 0; i < 3; i++ {
		_ = os.WriteFile(file, []byte(fmt.Sprintf("edit %d\n", i)), 0o644)
		time.Sleep(2 * lockTime)
	}

	reopened := dialWS(t, url)
	waitReload(t, reopened, file)
}

func TestFailedHandshakeDoesNotBreakReload(t *testing.T) {
	_, file, url := startWS(t)
	tab := dialWS(t, url)

	// A plain GET is not a WebSocket handshake.
	res, err := http.Get("http" + strings.TrimPrefix(url, "ws"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("plain GET: got status %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
	// A handshake from another origin is rejected.
	_, res, err = websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"https://evil.example"}})
	if err == nil {
		t.Fatal("cross-origin handshake succeeded")
	}
	if res == nil {
		t.Fatalf("cross-origin handshake: no response: %v", err)
	}
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-origin handshake: got status %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	waitReload(t, tab, file)
}

func TestCloseFrameIsAnswered(t *testing.T) {
	_, _, url := startWS(t)
	c := dialWS(t, url)

	_ = c.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := c.ReadMessage()
	if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Errorf("after sending a close frame, got %v, want the server's close frame", err)
	}
}

// count returns the number of registered clients.
func (h *hub) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func TestClosedClientsAreRemoved(t *testing.T) {
	handler, _, url := startWS(t)
	h := handler.(*hub)

	const n = 10
	conns := make([]*websocket.Conn, n)
	for i := range conns {
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatal(err)
		}
		conns[i] = c
	}
	if got := h.count(); got != n {
		t.Fatalf("%d clients registered, want %d", got, n)
	}
	for i, c := range conns {
		if i%2 == 0 {
			// Closing a tab sends a close frame; a crash or a lost network does not.
			_ = c.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, ""), time.Now().Add(time.Second))
		}
		_ = c.Close()
	}
	deadline := time.Now().Add(5 * time.Second)
	for h.count() > 0 {
		if time.Now().After(deadline) {
			t.Fatalf("%d of %d closed clients are still registered", h.count(), n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func receiveNotification(t *testing.T, sent <-chan struct{}) {
	t.Helper()
	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("no notification was sent")
	}
}

func TestHubCoalescesPendingNotifications(t *testing.T) {
	h := newHub()
	c := newClient()
	h.add(c)

	sent := make(chan struct{}, 10)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	stopped := make(chan struct{})
	held := false
	go func() {
		defer close(stopped)
		c.run(func() error {
			sent <- struct{}{}
			if !held {
				// Hold the first send, like a client that is slow to read.
				held = true
				<-release
			}
			return nil
		}, func() error { return nil })
	}()
	defer func() {
		unblock()
		close(c.done)
		<-stopped
	}()

	h.notify()
	receiveNotification(t, sent)
	// The writer is busy now, so these must be merged into one, not dropped.
	for i := 0; i < 5; i++ {
		h.notify()
	}
	unblock()
	receiveNotification(t, sent)
	// Nothing else is pending: the merged notifications are sent only once.
	select {
	case <-sent:
		t.Fatal("merged notifications were sent more than once")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestHubConcurrentUse(t *testing.T) {
	h := newHub()
	nop := func() error { return nil }
	var writers sync.WaitGroup

	// These clients stay registered while others come and go.
	stayed := make([]*client, 4)
	reached := make([]chan struct{}, len(stayed))
	for i := range stayed {
		c := newClient()
		got := make(chan struct{}, 1)
		stayed[i], reached[i] = c, got
		h.add(c)
		writers.Add(1)
		go func() {
			defer writers.Done()
			c.run(func() error {
				select {
				case got <- struct{}{}:
				default:
				}
				return nil
			}, nop)
		}()
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.notify()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c := newClient()
				h.add(c)
				writers.Add(1)
				go func() {
					defer writers.Done()
					c.run(func() error { return nil }, nop)
				}()
				h.remove(c)
				close(c.done)
			}
		}()
	}
	wg.Wait()

	// Every client that stayed must still get a notification sent afterwards.
	for _, got := range reached {
		select {
		case <-got:
		default:
		}
	}
	h.notify()
	for i, got := range reached {
		select {
		case <-got:
		case <-time.After(5 * time.Second):
			t.Errorf("client %d did not get the notification sent after concurrent use", i)
		}
	}
	for _, c := range stayed {
		close(c.done)
	}
	writers.Wait()
}
