package watchf

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mgutz/ansi"
	"golang.org/x/net/context"
	"gopkg.in/fsnotify.v1"
)

type Runner interface {
	Run(...Action)
}

type BasicRunner struct {
	Context context.Context
}

func (r BasicRunner) Run(actions ...Action) {
	for _, e := range actions {
		select {
		case <-r.Context.Done():
			break
		default:
		}

		if op := e.Run(); op != Continue {
			break
		}
	}
}

type StepOp int

const (
	Halt StepOp = iota
	Continue
)

type Action interface {
	Run() StepOp
}

type cmdAction struct {
	command string
	event   fsnotify.Event
}

func (c cmdAction) Run() StepOp {
	start := time.Now()

	var cmd *exec.Cmd
	command := evaluate(c.command, c.event)
	args := strings.Fields(command)
	if len(args) > 1 {
		cmd = exec.Command(args[0], args[1:]...)
	} else {
		cmd = exec.Command(args[0])
	}

	out, err := cmd.CombinedOutput()
	elapsed := time.Now().Sub(start)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"elapsed": elapsed,
		}).Error(highlight(fmt.Sprintf("Run: %s", command), "red+b"))
	} else {
		log.WithFields(log.Fields{
			"elapsed": time.Now().Sub(start),
		}).Info(highlight(fmt.Sprintf("Run: %s", command), "cyan+b"))
	}
	if len(out) > 0 {
		fmt.Print(string(out))
	}

	if err != nil {
		return Halt
	}
	return Continue
}

func highlight(text string, color string) string {
	if !isTerminal() {
		return text
	}
	return ansi.Color(text, color)
}

func isTerminal() bool {
	return log.IsTerminal()
}

func evaluate(cmd string, evt fsnotify.Event) string {
	cmd = strings.Replace(cmd, "%f", evt.Name, -1)
	cmd = strings.Replace(cmd, "%t", opName(evt.Op), -1)
	return cmd
}

func opName(op fsnotify.Op) string {
	var buffer bytes.Buffer
	if op&fsnotify.Create == fsnotify.Create {
		_, _ = buffer.WriteString("|CREATE")
	}
	if op&fsnotify.Remove == fsnotify.Remove {
		_, _ = buffer.WriteString("|REMOVE")
	}
	if op&fsnotify.Write == fsnotify.Write {
		_, _ = buffer.WriteString("|WRITE")
	}
	if op&fsnotify.Rename == fsnotify.Rename {
		_, _ = buffer.WriteString("|RENAME")
	}
	if op&fsnotify.Chmod == fsnotify.Chmod {
		_, _ = buffer.WriteString("|CHMOD")
	}

	if buffer.Len() == 0 {
		return ""
	}
	return buffer.String()[1:]
}
