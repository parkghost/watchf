package main

import (
	"code.google.com/p/go.exp/fsnotify"
	"io"
	"log"
	"os/exec"
	"strings"
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

	log.Printf("exec: \"%s %s\"\n", cmd.Args[0], strings.Join(cmd.Args[1:], " "))
	err := cmd.Run()

	if err != nil {
		log.Printf("exec: \"%s %s\" failed, err: %s\n", cmd.Args[0], strings.Join(cmd.Args[1:], " "), err)
	}

	return err
}

func evaluateVariables(command string, evt *fsnotify.FileEvent) string {
	command = strings.Replace(command, VarFilename, evt.Name, -1)
	command = strings.Replace(command, VarEventType, getEventType(evt), -1)
	return command
}
