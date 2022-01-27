package cmd

import (
	_ "embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"text/template"
)

type TemplateParam struct {
	Body   string
	Host   string
	Reload bool
	Mode   string
}

type Server struct {
	port int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

//go:embed template.html
var htmlTemplate string

const defaultPort = 3333

func (server *Server) Serve(param *Param) {
	port := defaultPort
	if server.port > 0 {
		port = server.port
	}

	filename := targetFile(param.filename)
	dir := filepath.Dir(filename)

	r := http.NewServeMux()
	r.Handle("/__/md", wrapHandler(mdHandler(filename)))
	r.Handle("/ws", wsHandler(filename))
	rootHandler := handler(filename, param, http.FileServer(http.Dir(dir)))
	r.Handle("/", wrapHandler(rootHandler))

	port = getPort(port)

	logInfo("Accepting connections at http://*:%d/\n", port)

	if param.autoOpen {
		logInfo("Open http://*:%d/ on your browser\n", port)
		go openBrowser(port)
	}

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), r)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

func handler(filename string, param *Param, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tmpl, err := template.New("HTML Template").Parse(htmlTemplate)
		if err != nil {
			logInfo("error:%v", err)
			http.NotFound(w, r)
			return
		}

		markdown := slurp(filename)
		html := toHTML(markdown)

		modeString := getModeString(param.forceLightMode, param.forceDarkMode)

		param := TemplateParam{Body: html, Host: r.Host, Reload: param.reload, Mode: modeString}

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

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func wrapHandler(wrappedHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := NewLoggingResponseWriter(w)
		wrappedHandler.ServeHTTP(lrw, r)

		statusCode := lrw.statusCode
		logInfo("%s [%d] %s", r.Method, statusCode, r.URL)
	})
}

func getModeString(lightMode, darkMode bool) string {
	if lightMode {
		return "light"
	} else if darkMode {
		return "dark"
	}
	return ""
}

func getPort(port int) int {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logInfo("http://*:%d/ is already used\n", port)
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("Can not find an available port: %v", err)
		}
	}
	port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}
