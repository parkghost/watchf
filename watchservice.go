package main

import (
	"code.google.com/p/go.exp/fsnotify"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	EventBufSize = 1024 * 1024
)

type WatchService struct {
	path   string
	config *Config

	watcher              *fsnotify.Watcher
	watchFlags           uint32
	includePatternRegexp *regexp.Regexp

	executor *Executor

	dirs    map[string]bool
	entries map[string]*FileEntry
}

func NewWatchService(path string, config *Config) (service *WatchService, err error) {
	watchFlags, err := calculateWatchFlags(config.Events)
	if err != nil {
		return
	}

	includePatternRegexp, err := regexp.Compile(config.IncludePattern)
	if err != nil {
		return
	}

	service = &WatchService{
		path,
		config,
		nil,
		watchFlags,
		includePatternRegexp,
		&Executor{os.Stdout, os.Stderr},
		make(map[string]bool),
		make(map[string]*FileEntry),
	}
	return
}

func calculateWatchFlags(events []string) (watchFlags uint32, err error) {
	Logln("watching events:")
	for _, event := range events {
		found := false
		for _, item := range GeneralEventBits {
			if strings.ToLower(event) == strings.ToLower(item.Name) {
				Logf("  %s\n", item.Name)
				watchFlags = watchFlags | item.Value
				found = true
			}
		}

		if !found {
			err = errors.New(fmt.Sprintf("the event %s was not found", event))
			return
		}
	}

	return
}

var GeneralEventBits = []struct {
	Value uint32
	Name  string
	Desc  string
}{
	{fsnotify.FSN_ALL, "all", "Create/Delete/Modify/Rename"},
	{fsnotify.FSN_CREATE, "create", "File/directory created in watched directory"},
	{fsnotify.FSN_DELETE, "delete", "File/directory deleted from watched directory"},
	{fsnotify.FSN_MODIFY, "modify", "File was modified or Metadata changed"},
	{fsnotify.FSN_RENAME, "rename", "File moved out of watched directory"},
}

func (w *WatchService) Start() (err error) {
	events := make(chan *fsnotify.FileEvent, EventBufSize)
	w.startWatcher(events) // events producer
	w.startWorker(events)  // events consumer
	return
}

func (w *WatchService) startWatcher(events chan<- *fsnotify.FileEvent) (err error) {
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case evt, ok := <-w.watcher.Event:
				if ok {
					// emit events from watcher.Event to buffered channel in order to non-ignored events
					events <- evt
				} else {
					close(events)
					return
				}
			case err, ok := <-w.watcher.Error:
				if ok {
					log.Println("watcher err:", err)
				} else {
					return
				}
			}
		}
	}()

	err = w.watchFolders()
	return
}

func (w *WatchService) watchFolders() (err error) {
	if w.config.Recursive {
		err = filepath.Walk(w.path, func(path string, info os.FileInfo, errPath error) error {
			if info.IsDir() {
				relativePath := "./" + path
				if errPath == nil {
					w.dirs[relativePath] = true
					Logln("watching: ", relativePath)
					errWatcher := w.watcher.WatchFlags(path, w.watchFlags)
					if errWatcher != nil {
						return errWatcher
					}
				} else {
					log.Printf("skip dir %s, caused by: %s\n", relativePath, errPath)
					return filepath.SkipDir
				}
			}
			return nil
		})
	} else {
		err = w.watcher.WatchFlags(w.path, w.watchFlags)
	}
	return
}

func (w *WatchService) startWorker(events <-chan *fsnotify.FileEvent) {
	go func() {
		var lastExec time.Time
		for evt := range events {
			Logf("%s: %s", getEventType(evt), evt.Name)

			w.syncWatchersAndCaches(evt)

			if checkPatternMatching(w.includePatternRegexp, evt) {
				if checkExecInterval(lastExec, w.config.Interval, time.Now()) {
					if w.isDir(evt.Name) {
						lastExec = time.Now()
						w.run(evt)
					} else {
						// ignore file attributes changed
						if evt.IsModify() && !checkFileContentChanged(w.entries, evt.Name) {
							continue
						}
						lastExec = time.Now()
						w.run(evt)
					}
				} else {
					Logf("%s: %s dropped", getEventType(evt), evt.Name)
				}
			}
		}
	}()
}

func getEventType(evt *fsnotify.FileEvent) string {
	eventType := ""

	switch {
	case evt.IsCreate():
		eventType = "ENTRY_CREATE"
	case evt.IsModify():
		eventType = "ENTRY_MODIFY"
	case evt.IsDelete():
		eventType = "ENTRY_DELETE"
	case evt.IsRename():
		eventType = "ENTRY_RENAME"
	}
	return eventType
}

func (w *WatchService) syncWatchersAndCaches(evt *fsnotify.FileEvent) {
	path := evt.Name
	switch {
	case evt.IsCreate():
		stat, err := os.Stat(path)
		if err != nil {
			Logln(err)
		} else {
			if stat.IsDir() {
				Logln("watching: ", path)
				w.dirs[path] = true
				w.watcher.WatchFlags(path, w.watchFlags)
			}
		}

	case evt.IsRename(), evt.IsDelete():
		if w.isDir(path) {
			Logln("remove watching: ", path)
			delete(w.dirs, path)
			w.watcher.RemoveWatch(path)

			dirPath := path + string(os.PathSeparator)
			for entryPath, _ := range w.entries {
				if strings.HasPrefix(entryPath, dirPath) {
					delete(w.entries, entryPath)
				}
			}
		} else {
			delete(w.entries, path)
		}
	}
}

func (w *WatchService) isDir(path string) bool {
	_, ok := w.dirs[path]
	return ok
}

func (w *WatchService) run(evt *fsnotify.FileEvent) {
	for _, command := range w.config.Commands {
		err := w.executor.execute(command, evt)
		if err != nil && !ContinueOnError {
			break
		}
	}
}

func (w *WatchService) Stop() error {
	return w.watcher.Close()
}
