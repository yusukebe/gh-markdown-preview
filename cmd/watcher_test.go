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

// startFakeWatch runs watch on a watcher whose channels the test feeds
// directly, and returns the times at which notify was called.
func startFakeWatch(t *testing.T) (*fsnotify.Watcher, <-chan time.Time) {
	t.Helper()
	watcher := &fsnotify.Watcher{Events: make(chan fsnotify.Event), Errors: make(chan error)}
	notified := make(chan time.Time, 100)
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		watch(watcher, "README.md", func() { notified <- time.Now() })
	}()
	t.Cleanup(func() {
		close(watcher.Events)
		close(watcher.Errors)
		<-returned
	})
	return watcher, notified
}

// save reports a write to README.md and returns when watch received it.
func save(watcher *fsnotify.Watcher) time.Time {
	watcher.Events <- fsnotify.Event{Name: "README.md", Op: fsnotify.Write}
	return time.Now()
}

// waitNotifiedAfter waits for a notification sent after t0.
func waitNotifiedAfter(t *testing.T, notified <-chan time.Time, t0 time.Time) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case at := <-notified:
			if at.After(t0) {
				return
			}
		case <-timeout:
			t.Fatal("no notification after the last change")
		}
	}
}

func TestWatchNoChangeLost(t *testing.T) {
	watcher, notified := startFakeWatch(t)

	waitNotifiedAfter(t, notified, save(watcher))
	// Save again right after the reload, as a formatter does after a save.
	waitNotifiedAfter(t, notified, save(watcher))
}

func TestWatchNotifiesAfterBurst(t *testing.T) {
	watcher, notified := startFakeWatch(t)

	var last time.Time
	for i := 0; i < 5; i++ {
		last = save(watcher)
		time.Sleep(10 * time.Millisecond)
	}
	waitNotifiedAfter(t, notified, last)
}
