package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.exp/fsnotify"
	"errors"
	"fmt"
	"hash/adler32"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	FileBlockSize = 1 * 1024 * 1024
	EventBufSize  = 1024 * 1024
	WritingDelay  = time.Duration(100) * time.Millisecond
	VarFilename   = "%f"
	VarEventType  = "%t"
)

type WatchService struct {
	path   string
	config *Config

	watcher              *fsnotify.Watcher
	watchFlags           uint32
	dirs                 map[string]bool
	entries              map[string]*FileEntry
	includePatternRegexp *regexp.Regexp

	Stdout io.Writer
	Stderr io.Writer
}

func NewWatchService(path string, config *Config) (service *WatchService, err error) {
	watchFlags, err := getFlagsValue(config)
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
		make(map[string]bool),
		make(map[string]*FileEntry),
		includePatternRegexp,
		os.Stdout,
		os.Stderr,
	}
	return
}

func getFlagsValue(config *Config) (flagVar uint32, err error) {
	for _, item := range config.Events {
		found := false
		for _, event := range GeneralEventBits {
			if item == strings.ToLower(event.Name) {
				flagVar = flagVar | event.Value
				found = true
			}
		}

		if !found {
			err = errors.New(fmt.Sprintf("the event %s was not found", item))
			return
		}
	}

	verbose := func() {
		log.Println("watching events:")
		for _, event := range GeneralEventBits {
			if flagVar&event.Value == event.Value {
				fmt.Fprintf(os.Stderr, "  %s\n", event.Name)
			}
		}
	}

	logFunc(verbose)
	return
}

func (w *WatchService) Start() (err error) {
	ch := make(chan *fsnotify.FileEvent, EventBufSize)
	w.startWatcher(ch)
	w.startWorker(ch)
	return
}

func (w *WatchService) isDir(path string) bool {
	_, ok := w.dirs[path]
	return ok
}

func (w *WatchService) startWorker(ch <-chan *fsnotify.FileEvent) {
	go func() {
		var lastExec time.Time
		for evt := range ch {
			logf("%s: %s", getEventType(evt), evt.Name)

			w.updateWatcherAndEntries(evt)

			// rename event will create another create event, so just ignore it
			if evt.IsRename() {
				continue
			}

			if checkPatternMatching(w.includePatternRegexp, evt) && checkExecInterval(lastExec, w.config.Interval, time.Now()) {
				if w.isDir(evt.Name) {
					lastExec = time.Now()
					w.run(evt)
				} else if checkContentWasChanged(w.entries, evt) {
					lastExec = time.Now()
					w.run(evt)
				} else {
					logf("%s: %s dropped", getEventType(evt), evt.Name)
				}
			} else {
				logf("%s: %s dropped", getEventType(evt), evt.Name)

			}

		}
	}()
}

func (w *WatchService) startWatcher(ch chan<- *fsnotify.FileEvent) (err error) {
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case evt, ok := <-w.watcher.Event:
				if ok {
					// emit event to another buffered channel in order to non-block watcher.Event channel
					ch <- evt
				} else {
					close(ch)
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

	err = w.addWatcher()
	return
}

func (w *WatchService) addWatcher() (err error) {
	if w.config.Recursive {
		err = filepath.Walk(w.path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				relativePath := "./" + path
				if err == nil {

					w.dirs[relativePath] = true
					logln("watching: ", relativePath)
					errWatcher := w.watcher.WatchFlags(path, w.watchFlags)
					if errWatcher != nil {
						return errWatcher
					}
				} else {
					log.Printf("path: %s, err: %s\n", relativePath, err)
				}
			}
			return nil
		})
	} else {
		err = w.watcher.WatchFlags(w.path, w.watchFlags)
	}
	return
}

func (w *WatchService) updateWatcherAndEntries(evt *fsnotify.FileEvent) {
	path := evt.Name
	switch {
	case evt.IsCreate():
		stat, err := os.Stat(path)
		if err != nil {
			logln(err)
		} else {
			if stat.IsDir() {
				logln("watching: ", path)
				w.dirs[path] = true
				w.watcher.WatchFlags(path, w.watchFlags)
			}
		}

	case evt.IsRename(), evt.IsDelete():
		if w.isDir(path) {
			logln("remove watching: ", path)
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

func (w *WatchService) Stop() error {
	return w.watcher.Close()
}

func (w *WatchService) run(evt *fsnotify.FileEvent) {
	for _, command := range w.config.Commands {
		err := w.execute(command, evt)
		if err != nil && !ContinueOnError {
			break
		}
	}
}

func (w *WatchService) execute(command string, evt *fsnotify.FileEvent) (err error) {
	command = applyCustomVariable(command, evt)
	args := strings.Split(command, " ")

	var cmd *exec.Cmd

	if len(args) > 1 {
		cmd = exec.Command(args[0], args[1:]...)
	} else {
		cmd = exec.Command(args[0])
	}

	logf("exec: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))

	buffer := &bytes.Buffer{}
	cmd.Stderr = buffer
	cmd.Stdout = buffer

	if err = cmd.Run(); err != nil {
		log.SetOutput(w.Stderr)
		log.Printf("run \"%s %s\" failed, err: %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "), err)
		log.SetOutput(os.Stderr)
	}

	if len(buffer.Bytes()) > 0 {
		fmt.Fprintln(w.Stdout, string(buffer.Bytes()))
	}
	return
}

func verboseMsgWrapper(title string, fun func() bool) bool {
	logln("[" + title + "]")
	result := fun()
	logf("[RESULT: %v]", result)

	return result
}

func checkPatternMatching(pattern *regexp.Regexp, evt *fsnotify.FileEvent) bool {
	return verboseMsgWrapper("check filename matching the pattern", func() bool {
		logf("%s ~= %s", pattern, evt.Name)
		matched := pattern.MatchString(evt.Name)
		return matched
	})

}

func checkExecInterval(lastExec time.Time, interval time.Duration, now time.Time) bool {
	return verboseMsgWrapper("check execution interval", func() bool {
		if interval == 0 {
			return true
		}
		nextExec := lastExec.Add(interval)
		delta := now.Sub(nextExec)
		logf("next execution time: %s, now: %s\n, delta:%s", nextExec, now, delta)
		return delta > 0
	})
}

func checkContentWasChanged(entries map[string]*FileEntry, evt *fsnotify.FileEvent) bool {
	return verboseMsgWrapper("check content was changed", func() bool {
		path := evt.Name

		if evt.IsModify() {
			entry, ok := entries[path]
			stat, err := os.Stat(path)
			if err != nil {
				return false
			}

			if stat.IsDir() {
				return false
			}

			if !ok {
				if newEntry, err := newFileEntry(path); err != nil {
					log.Println(err)
					return false
				} else {
					entries[path] = newEntry
				}
			} else {
				contentSize, err := getFilesizeWithRetry(path)
				logf("content size: %d, err: %v", contentSize, err)
				if err != nil {
					log.Println(err)
					return false
				}

				if entry.size != contentSize {
					entry.size = contentSize
				}

				contentHash, err := getContentHash(path)
				logf("content hash: %d, err: %v", contentHash, err)
				if err != nil {
					log.Println(err)
					return false
				}

				if entry.hash != contentHash {
					entry.hash = contentHash
				} else {
					return false
				}
			}

		}
		return true
	})
}

func applyCustomVariable(command string, evt *fsnotify.FileEvent) string {
	command = strings.Replace(command, VarFilename, evt.Name, -1)
	command = strings.Replace(command, VarEventType, getEventType(evt), -1)
	return command
}

type FileEntry struct {
	size int64
	hash uint32
}

func newFileEntry(filename string) (entry *FileEntry, err error) {
	contentSize, err := getFilesizeWithRetry(filename)
	if err != nil {
		return
	}

	sum, err := getContentHash(filename)
	if err != nil {
		return
	}

	entry = &FileEntry{contentSize, sum}
	return
}

func getFilesizeWithRetry(path string) (contentSize int64, err error) {
	contentSize, err = getFilesize(path)
	if err != nil {
		log.Println(err)
		return
	}

	if contentSize > 0 {
		return
	}

	//fallback
	time.Sleep(WritingDelay)
	contentSize, err = getFilesize(path)
	logf("[Fallback]content size: %d, err: %v", contentSize, err)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func getFilesize(filename string) (size int64, err error) {
	st, err := os.Stat(filename)
	if err != nil {
		return
	} else {
		size = st.Size()
	}

	return
}

func getContentHash(filename string) (sum uint32, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return
	}

	reader := bufio.NewReader(f)
	block := make([]byte, FileBlockSize)
	hash := adler32.New()

	size, errRead := reader.Read(block)
	for errRead == nil {
		hash.Write(block[:size])
		size, errRead = reader.Read(block)
	}
	if errRead != io.EOF {
		err = errRead
		return
	}

	sum = hash.Sum32()
	return
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
