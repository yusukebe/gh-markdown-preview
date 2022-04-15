package cmd

import (
	"regexp"
	"time"

	"github.com/fsnotify/fsnotify"
)

const ignorePattern = `\.swp$|~$|^\.DS_Store$|^4913$`
const lockTime = 100 * time.Millisecond

func createWatcher(dir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return watcher, err
	}
	logInfo("Watching %s/ for changes", dir)
	err = watcher.Add(dir)
	return watcher, err
}

func watch(done <-chan interface{}, errorChan chan<- error, reload chan<- bool, watcher *fsnotify.Watcher) {
	isLocked := false
	for {
		select {
		case event := <-watcher.Events:
			if isLocked {
				break
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r := regexp.MustCompile(ignorePattern)
				if r.MatchString(event.Name) {
					logDebug("Debug [ignore]: `%s`", event.Name)
				} else {
					logInfo("Change detected in %s, refreshing", event.Name)
					isLocked = true
					reload <- true
					timer := time.NewTimer(lockTime)
					go func() {
						<-timer.C
						isLocked = false
					}()
				}
			}
		case err := <-watcher.Errors:
			errorChan <- err
		case <-done:
			return
		}
	}
}
