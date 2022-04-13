package cmd

import (
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var socket *websocket.Conn

func wsHandler(watcher *fsnotify.Watcher) http.Handler {
	reload := make(chan bool, 1)
	errorChan := make(chan error)
	done := make(chan interface{})

	go watch(done, errorChan, reload, watcher)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		socket, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				logDebug("Debug [handshake error]: %s", err)
			}
			return
		}
		socket.SetReadDeadline(time.Now().Add(pongWait))
		socket.SetPongHandler(func(string) error { socket.SetReadDeadline(time.Now().Add(pongWait)); return nil })

		go wsReader(done, errorChan)
		go wsWriter(done, errorChan, reload)

		err = <-errorChan
		close(done)
		logInfo("Close WebSocket: %v\n", err)
		socket.Close()
	})
}

func wsReader(done <-chan interface{}, errorChan chan<- error) {
	for range done {
		_, _, err := socket.ReadMessage()
		if err != nil {
			logDebug("Debug [read message]: %s", err)
			errorChan <- err
		}
	}
}

func wsWriter(done <-chan interface{}, errChan chan<- error, reload <-chan bool) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-reload:
			err := socket.WriteMessage(websocket.TextMessage, []byte("reload"))
			if err != nil {
				logDebug("Debug [reload error]: %v", err)
				errChan <- err
			}
		case <-ticker.C:
			logDebug("Debug [ping send]: ping to client")
			err := socket.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				logDebug("Debug [ping error]: %v", err)
				// Do nothing
			}
		case <-done:
			return
		}
	}
}
