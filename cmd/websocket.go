package cmd

import (
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func wsHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				logDebug("Info: %s\n", err)
			}
			return
		}

		defer ws.Close()

		go wsWriter(ws, dir)

		wsReader(ws)
	})
}

func wsReader(ws *websocket.Conn) {
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			logDebug("Info: %s\n", err)
			break
		}
	}
}

func wsWriter(ws *websocket.Conn, dir string) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ws.Close()
		ticker.Stop()
	}()

	logInfo("Watching %s for changes", dir)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error:%v", err)
	}

	err = watcher.Add(dir)
	if err != nil {
		log.Fatalf("Error:%v", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logDebug("%v", event.Name)
				break
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r := regexp.MustCompile(`\.swp$|~$`)
				if r.MatchString(event.Name) {
					logInfo("Info: %s", event.Name)
					break
				}

				logInfo("Change detected in %s, refreshing", event.Name)
				err := ws.WriteMessage(websocket.TextMessage, []byte("reload"))
				if err != nil {
					logDebug("Info: %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Fatalf("Error:%v", err)
		case <-ticker.C:
			logDebug("%s\n", "Pinging client")
			err := ws.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				logDebug("Info: %v", err)
			}
		}
	}
}
