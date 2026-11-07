package cmd

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const lockTime = 100 * time.Millisecond

var reloadExts = map[string]bool{
	".md":   true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".webp": true,
}

var watchedDirs sync.Map

func createWatcher(dir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return watcher, err
	}
	logInfo("Watching %s/ for changes", dir)
	err = watcher.Add(dir)
	if err == nil {
		watchedDirs.Store(dir, struct{}{})
	}
	return watcher, err
}

// watchDir starts watching dir if it is not watched yet. It is used to pick
// up markdown files in subdirectories that are opened via links.
func watchDir(watcher *fsnotify.Watcher, dir string) {
	if watcher == nil {
		return
	}
	if _, loaded := watchedDirs.LoadOrStore(dir, struct{}{}); loaded {
		return
	}
	if err := watcher.Add(dir); err != nil {
		logInfo("Warn: failed to watch %s/: %v", dir, err)
		watchedDirs.Delete(dir)
		return
	}
	logInfo("Watching %s/ for changes", dir)
}

// shouldReload reports whether name is the previewed file, a markdown file
// that may be opened via a link, or an image. Changes to other files, such as
// a log file this command's output is redirected to, do not affect the preview.
func shouldReload(name, filename string) bool {
	if filename != "" && filepath.Clean(name) == filepath.Clean(filename) {
		return true
	}
	return reloadExts[strings.ToLower(filepath.Ext(name))]
}

// watch calls notify for every change that should reload the preview, until
// the watcher is closed. Errors are only logged: they must not stop live
// reloading.
func watch(watcher *fsnotify.Watcher, filename string, notify func()) {
	isLocked := false
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if isLocked {
				break
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if !shouldReload(event.Name, filename) {
					logDebug("Debug [ignore]: `%s`", event.Name)
				} else {
					logInfo("Change detected in %s, refreshing", event.Name)
					isLocked = true
					notify()
					timer := time.NewTimer(lockTime)
					go func() {
						<-timer.C
						isLocked = false
					}()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logInfo("Warn: %v", err)
		}
	}
}
