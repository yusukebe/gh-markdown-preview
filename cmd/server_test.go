package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	filename := "../testdata/markdown-demo.md"
	dir := filepath.Dir(filename)
	param := &Param{
		reload: false,
	}
	ts := httptest.NewServer(handler(filename, param, http.FileServer(http.Dir(dir)), nil))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("server status error, got: %v", res.StatusCode)
	}
	if res.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("content type error, got: %s\n", res.Header.Get("Content-Type"))
	}

	r2, err := http.Get(ts.URL + "/images/dinotocat.png")
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	if r2.StatusCode != 200 {
		t.Errorf("server status error, got: %v", res.StatusCode)
	}
	if r2.Header.Get("Content-Type") != "image/png" {
		t.Errorf("content type error, got: %s\n", r2.Header.Get("Content-Type"))
	}

}

func TestMdHandler(t *testing.T) {
	filename := "../testdata/markdown-demo.md"
	param := &Param{reload: false}
	ts := httptest.NewServer(mdHandler(filename, param))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("server status error, got: %v", res.StatusCode)
	}
	if res.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("content type error, got: %s\n", res.Header.Get("Content-Type"))
	}
}

func TestWrapHandler(t *testing.T) {
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := NewLoggingResponseWriter(w)
		wrappedHandler.ServeHTTP(lrw, r)
		statusCode := lrw.statusCode

		// XXX
		if statusCode != 200 {
			t.Errorf("logging response status code error, got: %v", statusCode)
		}

	})

	ts := httptest.NewServer(handler)
	defer ts.Close()
	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("server status error, got: %v", res.StatusCode)
	}
}

func TestGetModeString(t *testing.T) {
	modeString := getModeString(true, false)
	expected := "light"
	if modeString != expected {
		t.Errorf("mode string is not: %s", modeString)
	}

	modeString = getModeString(true, false)
	expected = "light"
	if modeString != expected {
		t.Errorf("mode string is not: %s", modeString)
	}

	modeString = getModeString(false, false)
	expected = ""
	if modeString != expected {
		t.Errorf("mode string is not: %s", modeString)
	}
}

func TestMdHandlerMissingFile(t *testing.T) {
	filename := "../testdata/markdown-demo.md"
	param := &Param{reload: false}
	ts := httptest.NewServer(mdHandler(filename, param))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/?path=missing.md")
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 404 {
		t.Errorf("server status error, got: %v", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if strings.Contains(string(body), "testdata") {
		t.Errorf("404 response should not include the file path, got: %s", body)
	}
}

func TestHandlerLinkedMarkdown(t *testing.T) {
	filename := "../testdata/markdown-demo.md"
	dir := filepath.Dir(filename)
	param := &Param{reload: false}
	ts := httptest.NewServer(handler(filename, param, http.FileServer(http.Dir(dir)), nil))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/gfm-alerts.md")
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("server status error, got: %v", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "<title>gfm-alerts.md</title>") {
		t.Errorf("linked markdown was not rendered, got: %s", body)
	}

	r2, err := http.Get(ts.URL + "/missing.md")
	if err != nil {
		t.Fatalf("unexpected: %v\n", err)
	}
	if r2.StatusCode != 404 {
		t.Errorf("server status error, got: %v", r2.StatusCode)
	}
}

func TestResolveMarkdownPath(t *testing.T) {
	filename := filepath.Join("docs", "README.md")
	tests := []struct {
		urlPath  string
		expected string
	}{
		{"", filename},
		{"/", filename},
		{"/guide.md", filepath.Join("docs", "guide.md")},
		{"sub/page.md", filepath.Join("docs", "sub", "page.md")},
		{"/../../etc/passwd.md", filepath.Join("docs", "etc", "passwd.md")},
		{"..\\..\\etc\\passwd.md", filepath.Join("docs", "etc", "passwd.md")},
		{"/sub\\..\\..\\secret.md", filepath.Join("docs", "secret.md")},
	}
	for _, tt := range tests {
		if got := resolveMarkdownPath(filename, tt.urlPath); got != tt.expected {
			t.Errorf("resolveMarkdownPath(%q) = %q, want %q", tt.urlPath, got, tt.expected)
		}
	}
}
