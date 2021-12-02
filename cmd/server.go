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
	Body   string
	Host   string
	Reload bool
}

type Server struct {
	port int
}

//go:embed template.html
var htmlTemplate string

const defaultPort = 3333

func (server *Server) Serve(filename string, reload bool) {
	port := defaultPort
	if server.port > 0 {
		port = server.port
	}
	log.Printf("Accepting connections at http://*:%d/\n", port)

	filename = targetFile(filename)
	dir := filepath.Dir(filename)

	r := http.NewServeMux()
	r.Handle("/__/md", wrapHandler(mdHandler(filename)))
	r.Handle("/ws", wsHandler(filename))
	rootHandler := handler(filename, reload, http.FileServer(http.Dir(dir)))
	r.Handle("/", wrapHandler(rootHandler))

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), r)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func handler(filename string, reload bool, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		param := TemplateParam{Body: html, Host: r.Host, Reload: reload}

		if err := tmpl.Execute(w, param); err != nil {
			log.Fatalf("error:%v", err)
		}
	})
}

func mdHandler(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		markdown := slurp(filename)
		html := toHTML(markdown)

		fmt.Fprintf(w, "%s", html)
	})
}
