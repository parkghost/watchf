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

type Handler interface {
	Handle(context.Context, fsnotify.Event)
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
}

func New(ctx context.Context, cfg *config.Config, path string, handler Handler) (*WatchService, error) {
	ws := new(WatchService)
	ws.path = path
	ws.recursive = cfg.Recursive
	ws.excludeRE = cfg.ExcludePattern.Regexp
	ws.flags = flags(cfg.Events)
	ws.handler = handler
	ws.ctx, ws.cancelFn = context.WithCancel(ctx)

	return ws, nil
}

const allOps = fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename | fsnotify.Chmod

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
			flags = allOps
		default:
			panic("invalid event:" + event)
		}
	}

	return flags
}

func (ws *WatchService) Start() error {
	log.Debug("starting WatchService")
	select {
	case <-ws.ctx.Done():
		return errors.New("WatchService closed")
	default:
	}

	err := ws.init()
	if err != nil {
		return fmt.Errorf("inital error: %s", err)
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
		err = ws.addSubFolders()
		if err != nil {
			return err
		}
		return nil
	}

	log.Debugf("watching: %s", ws.path)
	err = ws.watcher.Add(ws.path)
	if err != nil {
		return err
	}

	return nil
}

func (ws *WatchService) addSubFolders() error {
	return filepath.Walk(ws.path, func(path string, info os.FileInfo, errPath error) error {
		if info.IsDir() {
			if errPath != nil {
				log.WithField("error", errPath).Debugf("skipped dir %s", path)
				return filepath.SkipDir
			}

			if path != "." && ws.excludeRE.MatchString(path) {
				log.Debugf("skipped dir %s", path)
				return filepath.SkipDir
			}

			log.Debugf("watching: %s", path)
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
				log.Fatalf("watcher err: %s", err)
			}
			return
		}
	}
}

func (ws *WatchService) dispatch(evt fsnotify.Event) {
	if ws.flags&evt.Op == 0 {
		log.Debugf("skipped event: %s %s", opName(evt.Op), evt.Name)
		return
	}
	log.Infof("New event: %s %s", opName(evt.Op), evt.Name)

	ws.handler.Handle(ws.ctx, evt)
}

func (ws *WatchService) Stop() error {
	log.Debug("stopping WatchService")
	ws.cancelFn()

	return ws.watcher.Close()
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
		log.WithField("since", since).Debugf("limited event: %s %s", opName(evt.Op), evt.Name)
		return
	}

	filename := filepath.Base(evt.Name)
	included := h.includeRE.MatchString(filename)
	log.WithField("match", included).Debugf("check includePattern %s ~= %s ", h.includeRE, filename)
	if !included {
		return
	}

	excluded := h.excludeRE.MatchString(filename)
	log.WithField("match", excluded).Debugf("check excludePattern %s ~= %s ", h.excludeRE, filename)
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
	log.Debugf("actions: %v", strings.Join(h.commands, " > "))

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
