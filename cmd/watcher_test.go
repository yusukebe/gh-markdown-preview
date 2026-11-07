package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestShouldReload(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"README.md", "README.md", true},
		{"./README.md", "README.md", true},
		{"docs/README", "docs/README", true},
		{"docs/guide.md", "README.md", true},
		{"images/logo.png", "README.md", true},
		{"images/photo.JPG", "README.md", true},
		{"images/diagram.svg", "README.md", true},
		{"server.log", "README.md", false},
		{"README.md.swp", "README.md", false},
		{"README.md~", "README.md", false},
		{"4913", "README.md", false},
		{"server.log", "", false},
	}
	for _, tt := range tests {
		actual := shouldReload(tt.name, tt.filename)
		if actual != tt.expected {
			t.Errorf("shouldReload(%q, %q) = %v, want %v", tt.name, tt.filename, actual, tt.expected)
		}
	}
}

func TestWatchKeepsGoingAfterError(t *testing.T) {
	watcher := &fsnotify.Watcher{Events: make(chan fsnotify.Event), Errors: make(chan error)}
	notified := make(chan struct{}, 1)
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		watch(watcher, "README.md", func() { notified <- struct{}{} })
	}()

	watcher.Errors <- errors.New("event queue overflow")
	watcher.Events <- fsnotify.Event{Name: "README.md", Op: fsnotify.Write}
	select {
	case <-notified:
	case <-time.After(5 * time.Second):
		t.Fatal("a change after a watcher error was not notified")
	}

	close(watcher.Events)
	close(watcher.Errors)
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not return after the watcher was closed")
	}
}
