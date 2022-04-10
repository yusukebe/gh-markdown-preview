package cmd

import (
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	pongWait      = 60 * time.Second
	pingPeriod    = (pongWait * 9) / 10
	ignorePattern = `\.swp$|~$|^\.DS_Store$|^4913$`
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var reload chan bool
var socket *websocket.Conn

func watch(dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error:%v", err)
	}
	logInfo("Watching %s/ for changes", dir)
	err = watcher.Add(dir)
	if err != nil {
		log.Fatalf("Error:%v", err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r := regexp.MustCompile(ignorePattern)
				if r.MatchString(event.Name) {
					logDebug("Debug [ignore]: `%s`", event.Name)
					break
				}
				logInfo("Change detected in %s, refreshing", event.Name)
				reload <- true
			}
		case err := <-watcher.Errors:
			log.Fatalf("Error:%v", err)
		}
	}
}

func wsHandler(dir string) http.Handler {
	go watch(dir)
	reload = make(chan bool, 1)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		socket, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				logDebug("Debug [handshake error]: %s", err)
			}
			return
		}
		defer socket.Close()
		go wsWriter()
		wsReader()
	})
}

func wsReader() {
	socket.SetReadDeadline(time.Now().Add(pongWait))
	socket.SetPongHandler(func(string) error { socket.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := socket.ReadMessage()
		if err != nil {
			logDebug("Debug [read message]: %s", err)
			break
		}
	}
	socket.Close()
}

func wsWriter() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		socket.Close()
	}()

	for {
		select {
		case <-reload:
			err := socket.WriteMessage(websocket.TextMessage, []byte("reload"))
			if err != nil {
				logDebug("Debug [reload error]: %v", err)
				close(reload)
				return
			}
		case <-ticker.C:
			logDebug("Debug [ping send]: ping to client")
			err := socket.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				logDebug("Debug [ping error]: %v", err)
			}
		}
	}
}
