package cmd

import (
	"regexp"

	"github.com/fsnotify/fsnotify"
)

const ignorePattern = `\.swp$|~$|^\.DS_Store$|^4913$`

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
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r := regexp.MustCompile(ignorePattern)
				if r.MatchString(event.Name) {
					logDebug("Debug [ignore]: `%s`", event.Name)
				} else {
					logInfo("Change detected in %s, refreshing", event.Name)
					reload <- true
				}
			}
		case err := <-watcher.Errors:
			errorChan <- err
		case <-done:
			return
		}
	}
}
