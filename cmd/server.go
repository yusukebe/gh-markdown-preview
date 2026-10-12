package cmd

import (
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type TemplateParam struct {
	Title  string
	Body   string
	Host   string
	Reload bool
	Mode   string
}

type Server struct {
	host string
	port int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

//go:embed template.html
var htmlTemplate string

const defaultPort = 3333

func (server *Server) Serve(param *Param) error {
	host := server.host
	port := defaultPort
	if server.port > 0 {
		port = server.port
	}

	// Use a empty filename for stdin
	filename := ""
	if !param.useStdin {
		var err error
		filename, err = targetFile(param.filename)
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(filename)

	r := http.NewServeMux()
	r.Handle("/", wrapHandler(handler(filename, param, http.FileServer(http.Dir(dir)))))
	r.Handle("/__/md", wrapHandler(mdHandler(filename, param)))

	watcher, err := createWatcher(dir)
	if err != nil {
		return err
	}
	r.Handle("/ws", wsHandler(watcher))

	port, err = getPort(host, port)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", host, port)

	logInfo("Accepting connections at http://%s/\n", address)

	if param.autoOpen {
		logInfo("Open http://%s/ on your browser\n", address)
		go openBrowser(fmt.Sprintf("http://%s/", address))
	}

	err = http.ListenAndServe(address, r)
	if err != nil {
		return err
	}

	return nil
}

func handler(filename string, param *Param, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.HasSuffix(r.URL.Path, ".md") && r.URL.Path != "/" {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tmpl, err := template.New("HTML Template").Parse(htmlTemplate)
		if err != nil {
			logInfo("Warn: %v", err)
			http.NotFound(w, r)
			return
		}

		target := resolveMarkdownPath(filename, r.URL.Path)

		var markdown string
		if param.useStdin && param.stdinContent != "" && target == "" {
			markdown = param.stdinContent
		} else {
			markdown, err = slurp(target)
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		html, err := toHTML(markdown, param)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		title := getTitle(target)
		modeString := getModeString(param.forceLightMode, param.forceDarkMode)

		param := TemplateParam{Title: title, Body: html, Host: r.Host, Reload: param.reload, Mode: modeString}
		tmpl.Execute(w, param)
	})
}

func mdResponse(w http.ResponseWriter, filename string, param *Param) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var markdown string
	var err error
	if param.useStdin && param.stdinContent != "" && filename == "" {
		markdown = param.stdinContent
	} else {
		markdown, err = slurp(filename)
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	html, err := toHTML(markdown, param)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Fprintf(w, "%s", html)

}

func mdHandler(filename string, param *Param) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mdResponse(w, resolveMarkdownPath(filename, r.URL.Query().Get("path")), param)
	})
}

func resolveMarkdownPath(filename, urlPath string) string {
	cleaned := path.Clean("/" + urlPath)
	if cleaned == "/" {
		return filename
	}
	return filepath.Join(filepath.Dir(filename), filepath.FromSlash(cleaned))
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

func getTitle(filename string) string {
	return filepath.Base(filename)
}

func getModeString(lightMode, darkMode bool) string {
	if lightMode {
		return "light"
	} else if darkMode {
		return "dark"
	}
	return ""
}

func getPort(host string, port int) (int, error) {
	var err error
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		logInfo(err.Error())
		listener, err = net.Listen("tcp", fmt.Sprintf("%s:0", host))
	}
	port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, err
}
