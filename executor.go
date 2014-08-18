package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/mgutz/ansi"
	"gopkg.in/fsnotify.v0"
)

const (
	VarFilename  = "%f"
	VarEventType = "%t"
)

type Executor struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (e *Executor) execute(command string, evt *fsnotify.FileEvent) error {
	command = evaluateVariables(command, evt)
	commandArgs := strings.Split(command, " ")

	var cmd *exec.Cmd
	if len(commandArgs) > 1 {
		cmd = exec.Command(commandArgs[0], commandArgs[1:]...)
	} else {
		cmd = exec.Command(commandArgs[0])
	}
	cmd.Stderr = e.Stderr
	cmd.Stdout = e.Stdout

	msg := fmt.Sprintf("exec: \"%s %s\"", cmd.Args[0], strings.Join(cmd.Args[1:], " "))
	log.Println(ansi.Color(msg, "cyan+b"))
	err := cmd.Run()

	if err != nil {
		msg := fmt.Sprintf("exec: \"%s %s\" failed, err: %s", cmd.Args[0], strings.Join(cmd.Args[1:], " "), err)
		log.Println(ansi.Color(msg, "red+b"))
	}

	return err
}

func evaluateVariables(command string, evt *fsnotify.FileEvent) string {
	command = strings.Replace(command, VarFilename, evt.Name, -1)
	command = strings.Replace(command, VarEventType, getEventType(evt), -1)
	return command
}
