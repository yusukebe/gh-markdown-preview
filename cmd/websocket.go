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

// notification is a set of notification kinds, so that the notifications a
// slow client has not received yet are merged instead of dropped.
type notification uint8

const notifyReload notification = 1 << iota

// client is one WebSocket connection.
type client struct {
	mu      sync.Mutex
	pending notification
	wake    chan struct{} // capacity 1: one wake-up covers everything pending
	done    chan struct{}
}

func newClient() *client {
	return &client{wake: make(chan struct{}, 1), done: make(chan struct{})}
}

// notify adds n to the pending notifications and wakes the writer. It never
// blocks.
func (c *client) notify(n notification) {
	c.mu.Lock()
	c.pending |= n
	c.mu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *client) take() notification {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := c.pending
	c.pending = 0
	return n
}

// run is the only writer of the connection. It sends the pending
// notifications, and a ping every pingPeriod, until done is closed or a write
// fails.
func (c *client) run(send func(notification) error, ping func() error) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.wake:
			if n := c.take(); n != 0 {
				if err := send(n); err != nil {
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

func (h *hub) notify(n notification) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.notify(n)
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
		c.run(func(notification) error {
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
			logInfo("Close WebSocket: %v\n", err)
			break
		}
	}
	close(c.done)
	conn.Close()
	<-stopped
}
