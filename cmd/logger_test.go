package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
