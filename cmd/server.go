package cmd

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
)

type TemplateParam struct {
	Body string
	Host string
}

type Server struct {
	port     int
	template string
}

//go:embed template.html
var htmlTemplate string

const defaultPort = 3333

func (server *Server) Serve(filename string) {
	port := defaultPort
	if server.port > 0 {
		port = server.port
	}
	log.Printf("accepting connections at http://*:%d/\n", port)

	filename = targetFile(filename)

	dir := filepath.Dir(filename)
	http.Handle("/md", mdHandler(filename))
	http.Handle("/ws", wsHandler(filename))
	http.Handle("/", handler(filename, http.FileServer(http.Dir(dir))))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func handler(filename string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s [%s] %s", r.RemoteAddr, r.Method, r.URL)

		if r.URL.Path != "/" {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tmpl, err := template.New("HTML Template").Parse(htmlTemplate)
		if err != nil {
			log.Printf("error:%v", err)
			http.NotFound(w, r)
			return
		}

		markdown := slurp(filename)
		html := toHTML(markdown)
		param := TemplateParam{Body: html, Host: r.Host}

		if err := tmpl.Execute(w, param); err != nil {
			log.Fatalf("error:%v", err)
		}
	})
}

func mdHandler(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s [%s] %s", r.RemoteAddr, r.Method, r.URL)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		markdown := slurp(filename)
		html := toHTML(markdown)

		fmt.Fprintf(w, html)
	})
}
