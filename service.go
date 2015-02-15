package watchf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/parkghost/watchf/config"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"gopkg.in/fsnotify.v1"
)

var ErrServiceClosed = errors.New("closed")

type Service interface {
	Start() error
	Stop() error
}

type WatchService struct {
	path      string
	recursive bool
	excludeRE *regexp.Regexp

	watcher *fsnotify.Watcher
	flags   fsnotify.Op

	handler  Handler
	ctx      context.Context
	cancelFn context.CancelFunc

	dirs map[string]struct{}
}

func (ws *WatchService) Start() error {
	select {
	case <-ws.ctx.Done():
		return ErrServiceClosed
	default:
	}

	err := ws.init()
	if err != nil {
		return fmt.Errorf("init: %s", err)
	}

	go ws.run()
	return nil
}

func (ws *WatchService) init() error {
	var err error
	ws.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if ws.recursive {
		err = ws.addSubFolders(ws.path)
		if err != nil {
			return err
		}
		return nil
	}

	log.Debugf("Watching: %s", ws.path)
	err = ws.watcher.Add(ws.path)
	if err != nil {
		return err
	}
	return nil
}

func (ws *WatchService) addSubFolders(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, errPath error) error {
		if info.IsDir() {
			if errPath != nil {
				log.WithField("error", errPath).Debugf("Skipped dir: %s", path)
				return filepath.SkipDir
			}

			if path != "." && ws.excludeRE.MatchString(path) {
				log.Debugf("Skipped dir: %s", path)
				return filepath.SkipDir
			}

			log.Debugf("Watching: %s", path)
			ws.dirs[path] = struct{}{}
			err := ws.watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (ws *WatchService) run() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case evt, ok := <-ws.watcher.Events:
			if !ok {
				return
			}
			ws.dispatch(evt)
		case err := <-ws.watcher.Errors:
			if err != nil {
				log.Fatalf("Watcher err: %s", err)
			}
			return
		}
	}
}

func (ws *WatchService) dispatch(evt fsnotify.Event) {
	if ws.recursive && evt.Name != "." {
		ws.syncWatcher(evt)
	}

	if ws.flags&evt.Op == 0 {
		log.Debugf("Skipped event: %s %s", opName(evt.Op), evt.Name)
		return
	}

	log.Infof("New event: %s %s", opName(evt.Op), evt.Name)
	ws.handler.Handle(ws.ctx, evt)
}

func (ws *WatchService) syncWatcher(evt fsnotify.Event) {
	path := evt.Name
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	isDir, err := isDir(path, ws.dirs)
	if err != nil {
		log.Warnf("SyncWatcher(stat): %s", err)
		return
	}
	if !isDir {
		return
	}

	switch {
	case isCreate(evt):
		err := ws.addSubFolders(path)
		if err != nil {
			log.Warnf("SyncWatcher(scan): %s", err)
		}
	case isRemove(evt), isRename(evt):
		for ph := range ws.dirs {
			if strings.HasPrefix(ph, path) {
				log.Debugf("Stop watching: %s", ph)
				delete(ws.dirs, ph)
				err = ws.watcher.Remove(ph)
				if err != nil {
					log.Warnf("SyncWatcher(rm): %s", err)
				}
			}
		}
	}
}

func isDir(path string, cache map[string]struct{}) (bool, error) {
	if _, found := cache[path]; found {
		return true, nil
	}
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return f.IsDir(), nil
}

func isCreate(evt fsnotify.Event) bool {
	return evt.Op&fsnotify.Create == fsnotify.Create
}

func isRemove(evt fsnotify.Event) bool {
	return evt.Op&fsnotify.Remove == fsnotify.Remove
}

func isRename(evt fsnotify.Event) bool {
	return evt.Op&fsnotify.Rename == fsnotify.Rename
}

func (ws *WatchService) Stop() error {
	ws.cancelFn()
	return ws.watcher.Close()
}

func New(ctx context.Context, cfg *config.Config, path string, handler Handler) (Service, error) {
	ws := new(WatchService)
	ws.path = path
	ws.recursive = cfg.Recursive
	ws.excludeRE = cfg.ExcludePattern.Regexp
	ws.flags = flags(cfg.Events)
	ws.handler = handler
	ws.ctx, ws.cancelFn = context.WithCancel(ctx)
	if ws.recursive {
		ws.dirs = make(map[string]struct{})
	}
	return ws, nil
}

func flags(events []string) fsnotify.Op {
	var flags fsnotify.Op
	for _, event := range events {
		switch event {
		case "create":
			flags |= fsnotify.Create
		case "write":
			flags |= fsnotify.Write
		case "remove":
			flags |= fsnotify.Remove
		case "rename":
			flags |= fsnotify.Rename
		case "chmod":
			flags |= fsnotify.Chmod
		case "all":
			flags = fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename | fsnotify.Chmod
		default:
			panic("invalid event: " + event)
		}
	}
	return flags
}

type Handler interface {
	Handle(context.Context, fsnotify.Event)
}

type limitedHandler struct {
	includeRE *regexp.Regexp
	excludeRE *regexp.Regexp
	interval  time.Duration
	commands  []string

	nextExec time.Time
}

func (h *limitedHandler) Handle(ctx context.Context, evt fsnotify.Event) {
	if time.Now().Before(h.nextExec) {
		since := time.Now().Sub(h.nextExec.Add(-h.interval))
		log.WithField("since", since).Debugf("Limited event: %s %s", opName(evt.Op), evt.Name)
		return
	}

	filename := filepath.Base(evt.Name)
	included := h.includeRE.MatchString(filename)
	log.WithField("match", included).Debugf("Check includePattern %s ~= %s ", h.includeRE, filename)
	if !included {
		return
	}

	excluded := h.excludeRE.MatchString(filename)
	log.WithField("match", excluded).Debugf("Check excludePattern %s ~= %s ", h.excludeRE, filename)
	if excluded {
		return
	}

	h.runCmd(ctx, evt)

	h.nextExec = time.Now().Add(h.interval)
}

func (h *limitedHandler) runCmd(ctx context.Context, evt fsnotify.Event) {
	actions := make([]Action, 0, len(h.commands))
	for _, cmd := range h.commands {
		actions = append(actions, Action(cmdAction{cmd, evt}))
	}
	log.Debugf("Actions: %v", strings.Join(h.commands, " > "))

	runner := &BasicRunner{ctx}
	runner.Run(actions...)
}

func NewLimitedHandler(cfg *config.Config) Handler {
	h := new(limitedHandler)
	h.includeRE = cfg.IncludePattern.Regexp
	h.excludeRE = cfg.ExcludePattern.Regexp
	h.commands = cfg.Commands
	h.interval = cfg.Interval
	return h
}
