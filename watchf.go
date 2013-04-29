package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	Version         = "0.1.2"
	PidFile         = "watchf.pid"
	Program         = "watchf"
	ContinueOnError = false
)

func main() {
	// command line parsing
	var commands StringSet
	var stop bool
	var sensitive time.Duration

	flag.Var(&commands, "c", "Add arbitrary command(repeatable)")
	flag.BoolVar(&stop, "s", false, "To stop the "+Program+" Daemon")
	flag.DurationVar(&sensitive, "t", time.Duration(100)*time.Millisecond, "The time sensitive for avoid execute command frequently(time unit: ns/us/ms/s/m/h)")
	showVersion := flag.Bool("v", false, "show version")

	flag.Usage = func() {
		fmt.Println("Usage:\n  " + Program + " options 'pattern'")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("Variables:")
		fmt.Println("  $f: The filename of changed file")

		fmt.Println("Example 1:")
		fmt.Println("  " + Program + " -c 'go vet' -c 'go test' -c 'go install' '*.go'")
		fmt.Println("Example 2(Daemon):")
		fmt.Println("  " + Program + " -c 'chmod 644 $f' '*.exe' &")
		fmt.Println("  " + Program + " -s")
	}
	flag.Parse()

	pattern := os.Args[len(os.Args)-1]
	if pattern == "" {
		pattern = "*"
	}

	if *showVersion {
		fmt.Println("version " + Version)
	}

	if len(commands) == 0 && !stop {
		flag.Usage()
		os.Exit(-1)
	}

	// stop daemon via signal
	if stop {
		daemon := &Daemon{}
		daemon.Stop()
		return
	}

	// start daemon
	daemon := &Daemon{
		service: &WatchService{pattern, sensitive, commands, nil},
	}

	err := daemon.Start()
	checkError(err)

	// stop daemon
	waitForStop(daemon)
}

type Service interface {
	start() error
	stop() error
}

type Daemon struct {
	local   bool
	process *os.Process
	service Service
}

func (d *Daemon) Start() (err error) {
	if d.IsRunning() {
		log.Fatalln(Program + " is already running")
		return
	} else {
		err := ioutil.WriteFile(PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
		checkError(err)
		d.local = true
		return d.service.start()
	}
}

func (d *Daemon) Stop() (err error) {
	if d.IsRunning() {
		if d.local {
			os.Remove(PidFile)
			return d.service.stop()
		} else {
			err = d.process.Signal(os.Interrupt)
		}
	}
	return
}

func (d *Daemon) IsRunning() bool {
	if d.local {
		return true
	}

	pid, err := getDaemonPid()
	if err == nil {
		d.process, err = os.FindProcess(pid)
		return err == nil
	}
	return false
}

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func getDaemonPid() (pid int, err error) {
	data, err := ioutil.ReadFile(PidFile)
	if err == nil {
		pid, err = strconv.Atoi(string(data))
	}
	return
}

func waitForStop(daemon *Daemon) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Kill, os.Interrupt)

	<-quit
	if err := daemon.Stop(); err != nil {
		fmt.Printf(Program+" stop failed: %s\n", err)
	} else {
		fmt.Println(Program + " stopped")
	}
}

type WatchService struct {
	pattern   string
	sensitive time.Duration
	commands  []string
	watcher   *fsnotify.Watcher
}

const (
	TRUE  int32 = 1
	FALSE int32 = 0
)

func (w *WatchService) start() (err error) {
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
					// TODO: accept specific event
					if atomic.LoadInt32(&running) != TRUE && acceptedFile(w.pattern, evt) && lastExec.Add(w.sensitive).Before(now) {
						lastExec = now
						// using another goroutine to run command in order to non-blocking watcher.Event channel
						go execute(w.commands, evt, &running)
					}
				} else {
					break
				}
			case err, ok := <-w.watcher.Error:
				if ok {
					log.Fatalf("watcher error: %s\n", err)
				} else {
					break
				}
			}
		}
	}()

	// TODO: watching subdirectory
	err = w.watcher.Watch(".")
	return
}

func (w *WatchService) stop() error {
	return w.watcher.Close()
}

func acceptedFile(pattern string, ev *fsnotify.FileEvent) bool {
	matched, err := filepath.Match(pattern, ev.Name[2:])
	checkError(err)
	return matched
}

func execute(commands []string, evt *fsnotify.FileEvent, running *int32) {
	atomic.StoreInt32(running, TRUE)
	for _, command := range commands {
		command := applyCustomVariable(command, evt)
		// THINK: support command with pipeline

		args := strings.Split(command, " ")
		var cmd *exec.Cmd

		if len(args) > 1 {
			cmd = exec.Command(args[0], args[1:]...)
		} else {
			cmd = exec.Command(args[0])
		}

		if err := runCommand(cmd); err != nil && !ContinueOnError {
			break
		}
	}
	atomic.StoreInt32(running, FALSE)
}

func applyCustomVariable(command string, evt *fsnotify.FileEvent) string {
	return strings.Replace(command, "$f", evt.Name, -1)
}

func runCommand(cmd *exec.Cmd) (err error) {
	writer := &bytes.Buffer{}
	cmd.Stderr = writer
	cmd.Stdout = writer

	if err = cmd.Run(); err != nil {
		log.Printf("run \"%s\" failed, err: %s\n", strings.Join(cmd.Args, " "), err)
	}

	fmt.Println(string(writer.Bytes()))
	return
}

type StringSet []string

func (f *StringSet) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *StringSet) Set(value string) error {
	*f = append(*f, value)
	return nil
}
