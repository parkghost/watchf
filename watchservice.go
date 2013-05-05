package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.exp/fsnotify"
	"fmt"
	"hash/adler32"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	True                    int32 = 1
	False                   int32 = 0
	FileBlockSie                  = 1 * 1024 * 1024
	LenOfRelativePathPrefix       = 2
)

type WatchService struct {
	path      string
	pattern   string
	sensitive time.Duration
	commands  []string
	Stdout    io.Writer
	Stderr    io.Writer
	watcher   *fsnotify.Watcher
	entries   map[string]*FileEntry
}

func NewWatchService(path string, pattern string, sensitive time.Duration, commands []string) *WatchService {
	return &WatchService{path, pattern, sensitive, commands, os.Stdout, os.Stderr, nil, make(map[string]*FileEntry)}
}

func (w *WatchService) Start() (err error) {
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	go func() {
		var running = False
		var lastExec time.Time

		for {
			select {
			case evt, ok := <-w.watcher.Event:
				if ok {
					if verbose {
						log.Printf("%s: %s", getEventType(evt), evt.Name)
					}

					now := time.Now()

					if checkNoCommandRunning(&running) &&
						checkPatternMatching(w.pattern, evt) &&
						checkNextExecTimeHadExpired(lastExec, w.sensitive, now) &&
						checkContentWasChanged(w.entries, evt) {

						lastExec = now
						// using another goroutine to run command in order to non-blocking watcher.Event channel
						go w.run(evt, &running)
					}
				} else {
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

	// TODO: watching subdirectory
	if verbose {
		path := w.path
		if path == "." {
			path, _ = os.Getwd()
		}
		log.Println("watching: ", path)
	}
	err = w.watcher.Watch(w.path)
	return
}

func (w *WatchService) Stop() error {
	return w.watcher.Close()
}

func (w *WatchService) run(evt *fsnotify.FileEvent, running *int32) {
	atomic.StoreInt32(running, True)
	for _, command := range commands {
		err := w.execute(command, evt)
		if err != nil && !ContinueOnError {
			break
		}
	}
	atomic.StoreInt32(running, False)
}

func (w *WatchService) execute(command string, evt *fsnotify.FileEvent) (err error) {

	// THINK: support command with pipeline
	command = applyCustomVariable(command, evt)

	args := strings.Split(command, " ")
	var cmd *exec.Cmd

	if len(args) > 1 {
		cmd = exec.Command(args[0], args[1:]...)
	} else {
		cmd = exec.Command(args[0])
	}

	if verbose {
		log.Printf("exec: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

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
	if verbose {
		log.Println("[" + title + "]")
	}
	result := fun()
	if verbose {
		log.Printf("[RESULT: %v]", result)
	}
	return result
}

func checkNoCommandRunning(running *int32) bool {
	return verboseMsgWrapper("check no command running", func() bool {
		return atomic.LoadInt32(running) == False
	})
}

func checkPatternMatching(pattern string, evt *fsnotify.FileEvent) bool {
	return verboseMsgWrapper("check filename matching the pattern", func() bool {
		matched, err := filepath.Match(pattern, evt.Name[LenOfRelativePathPrefix:])
		if verbose {
			log.Printf("%s ~= %s", pattern, evt.Name[LenOfRelativePathPrefix:])
		}
		checkError(err)
		return matched
	})

}

func checkNextExecTimeHadExpired(lastExec time.Time, sensitive time.Duration, now time.Time) bool {
	return verboseMsgWrapper("check next execution time had expired", func() bool {
		nextExec := lastExec.Add(sensitive)
		result := nextExec.Before(now)
		if verbose {
			log.Printf("next execution time: %s, now: %s\n", nextExec, now)
		}
		return result
	})
}

func checkContentWasChanged(entries map[string]*FileEntry, evt *fsnotify.FileEvent) bool {
	return verboseMsgWrapper("check content was changed", func() bool {

		filename := evt.Name

		switch {
		case evt.IsCreate():
			if entry, err := newFileEntry(filename); err != nil {
				log.Println(err)
				return false
			} else {
				entries[filename] = entry
			}
		case evt.IsModify():
			entry, ok := entries[filename]

			if !ok {
				if entry, err := newFileEntry(filename); err != nil {
					log.Println(err)
					return false
				} else {
					entries[filename] = entry
				}
			} else {

				// THINK: wait for file closed
				contentSize, err := getFilesize(filename)
				if verbose {
					log.Printf("content size: %d, err: %v", contentSize, err)
				}
				if err != nil {
					log.Println(err)
					return false
				}

				if entry.size != contentSize {
					entry.size = contentSize
				}

				contentHash, err := getContentHash(filename)
				if verbose {
					log.Printf("content hash: %d, err: %v", contentHash, err)
				}
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

		case evt.IsDelete():
		case evt.IsRename():
			delete(entries, filename)
		}

		return true
	})

}

type FileEntry struct {
	size int64
	hash uint32
}

func newFileEntry(filename string) (entry *FileEntry, err error) {
	st, err := os.Stat(filename)
	st.Mode()
	if err != nil {
		return
	}

	sum, err := getContentHash(filename)
	if err != nil {
		return
	}

	entry = &FileEntry{st.Size(), sum}
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
	block := make([]byte, FileBlockSie)
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
		eventType = "CREATE"
	case evt.IsModify():
		eventType = "MODIFY"
	case evt.IsDelete():
		eventType = "DELETE"
	case evt.IsRename():
		eventType = "RENAME"
	}
	return eventType
}

func applyCustomVariable(command string, evt *fsnotify.FileEvent) string {
	command = strings.Replace(command, "$f", evt.Name, -1)
	command = strings.Replace(command, "$t", getEventType(evt), -1)

	return command
}
