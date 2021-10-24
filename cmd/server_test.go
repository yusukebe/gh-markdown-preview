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
	ts := httptest.NewServer(handler(filename, http.FileServer(http.Dir(dir))))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal("unexpected", err)
	}
	if res.StatusCode != 200 {
		t.Error("server status error")
	}
	if res.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("content type error\n")
	}

	r2, err := http.Get(ts.URL + "/images/dinotocat.png")
	if err != nil {
		t.Fatal("unexpected", err)
	}
	if r2.StatusCode != 200 {
		t.Errorf("server status error\n")
	}
	if r2.Header.Get("Content-Type") != "image/png" {
		t.Errorf("content type error\n")
	}

}
