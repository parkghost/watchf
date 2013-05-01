package main

import (
	"bytes"
	"fmt"
	"github.com/howeyc/fsnotify"
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
	TRUE  int32 = 1
	FALSE int32 = 0
)

type WatchService struct {
	path      string
	pattern   string
	sensitive time.Duration
	commands  []string
	watcher   *fsnotify.Watcher
	stdout    io.Writer
	stderr    io.Writer
}

func (w *WatchService) Start() (err error) {
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	go func() {
		var running = FALSE
		var lastExec time.Time

		for {
			select {
			case evt, ok := <-w.watcher.Event:
				if ok {
					now := time.Now()
					// TODO: verify file change by size and checksum
					if atomic.LoadInt32(&running) != TRUE && match(w.pattern, evt) && lastExec.Add(w.sensitive).Before(now) {
						lastExec = now
						// using another goroutine to run command in order to non-blocking watcher.Event channel
						go w.run(evt, &running)
					}
				} else {
					return
				}
			case err, ok := <-w.watcher.Error:
				if ok {
					checkError(err)
				} else {
					return
				}
			}
		}
	}()

	// TODO: watching subdirectory
	err = w.watcher.Watch(w.path)
	return
}

func (w *WatchService) Stop() error {
	return w.watcher.Close()
}

func (w *WatchService) run(evt *fsnotify.FileEvent, running *int32) {
	atomic.StoreInt32(running, TRUE)
	for _, command := range commands {
		err := w.execute(command, evt)
		if err != nil && !ContinueOnError {
			break
		}
	}
	atomic.StoreInt32(running, FALSE)
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

	buffer := &bytes.Buffer{}
	cmd.Stderr = buffer
	cmd.Stdout = buffer

	if err = cmd.Run(); err != nil {
		log.SetOutput(w.stderr)
		log.Printf("run \"%s\" failed, err: %s\n", strings.Join(cmd.Args, " "), err)
		log.SetOutput(os.Stderr)
	}

	if len(buffer.Bytes()) > 0 {
		fmt.Fprintln(w.stdout, string(buffer.Bytes()))
	}
	return
}

func match(pattern string, ev *fsnotify.FileEvent) bool {
	matched, err := filepath.Match(pattern, ev.Name[2:])
	checkError(err)
	return matched
}

func applyCustomVariable(command string, evt *fsnotify.FileEvent) string {
	command = strings.Replace(command, "$f", evt.Name, -1)

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
	command = strings.Replace(command, "$t", eventType, -1)

	return command
}
