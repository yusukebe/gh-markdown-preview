package cmd

import (
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// client is one WebSocket connection.
type client struct {
	mu      sync.Mutex
	pending bool          // a reload has not been sent yet; repeated reloads are merged
	wake    chan struct{} // capacity 1: one wake-up covers everything pending
	done    chan struct{}
}

func newClient() *client {
	return &client{wake: make(chan struct{}, 1), done: make(chan struct{})}
}

// notify marks a reload as pending and wakes the writer. It never blocks.
func (c *client) notify() {
	c.mu.Lock()
	c.pending = true
	c.mu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *client) take() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.pending
	c.pending = false
	return p
}

// run is the only writer of the connection. It calls send when a reload is
// pending, and ping every pingPeriod, until done is closed or a write fails.
// A failed ping also ends it: the connection is dead, so it is closed rather
// than kept registered.
func (c *client) run(send func() error, ping func() error) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.wake:
			if c.take() {
				if err := send(); err != nil {
					logDebug("Debug [reload error]: %v", err)
					return
				}
			}
		case <-ticker.C:
			logDebug("Debug [ping send]: ping to client")
			if err := ping(); err != nil {
				logDebug("Debug [ping error]: %v", err)
				return
			}
		case <-c.done:
			return
		}
	}
}

// hub keeps the connected clients and notifies every one of them.
type hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func newHub() *hub {
	return &hub{clients: make(map[*client]struct{})}
}

func (h *hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) remove(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *hub) notify() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.notify()
	}
}

func wsHandler(watcher *fsnotify.Watcher, filename string) http.Handler {
	h := newHub()
	go watch(watcher, filename, h.notify)
	return h
}

func (h *hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Register before the handshake completes, so that a change made right
	// after the page has connected is not missed.
	c := newClient()
	h.add(c)
	defer h.remove(c)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade has already replied with an HTTP error.
		if _, ok := err.(websocket.HandshakeError); !ok {
			logDebug("Debug [handshake error]: %s", err)
		}
		return
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		c.run(func() error {
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			return conn.WriteMessage(websocket.TextMessage, []byte("reload"))
		}, func() error {
			return conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
		})
		// After a failed write, this makes the read loop below return.
		conn.Close()
	}()

	// Reading is what answers the client's close frame and handles pongs.
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			logInfo("Close WebSocket: %v", err)
			break
		}
	}
	close(c.done)
	conn.Close()
	<-stopped
}
