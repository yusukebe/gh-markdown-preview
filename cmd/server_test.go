package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHandler(t *testing.T) {
	filename := "../testdata/markdown-demo.md"
	dir := filepath.Dir(filename)
	ts := httptest.NewServer(handler(filename, false, http.FileServer(http.Dir(dir))))
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
	ts := httptest.NewServer(mdHandler(filename))
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
