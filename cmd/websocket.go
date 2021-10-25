package cmd

import (
	"log"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wsHandler(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Println(err)
			}
			return
		}
		go wsWriter(ws, filename)
		wsReader(ws)
	})
}

func wsReader(ws *websocket.Conn) {
	defer ws.Close()
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func wsWriter(ws *websocket.Conn, filename string) {
	defer ws.Close()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("error:%v", err)
	}
	err = watcher.Add(filename)

	if err != nil {
		log.Fatalf("error:%v", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Modified file:", event.Name)
				if err := ws.WriteMessage(websocket.TextMessage, []byte("reload")); err == nil {
					return
				}
				log.Fatalf("error:%v", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Fatalf("error:%v", err)
		}
	}
}
