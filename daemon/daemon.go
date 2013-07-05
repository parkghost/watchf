package daemon

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

type Daemon struct {
	name       string
	pid        int
	foreground bool
	running    bool
	service    Service
}

type Service interface {
	Start() error
	Stop() error
}

func NewDaemon(name string, service Service) *Daemon {
	return &Daemon{name: name, service: service}
}

func (d *Daemon) Start() (err error) {
	if d.IsRunning() {
		return errors.New(d.name + " is already running")
	}
	if err = ioutil.WriteFile(d.getPidFilename(), []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return
	}

	if err = d.service.Start(); err != nil {
		return err
	}
	d.foreground = true
	d.running = true
	d.pid = os.Getpid()
	return
}

func (d *Daemon) getPidFilename() string {
	return "." + d.name + ".pid"
}

func (d *Daemon) IsRunning() bool {
	if d.running {
		return true
	}

	var err error
	d.pid, err = d.readPidFromFile(d.getPidFilename())
	if err == nil {
		d.running = isOSProcessRunning(d.pid)
	}

	return d.running
}

func (d *Daemon) readPidFromFile(filename string) (pid int, err error) {
	data, err := ioutil.ReadFile(filename)
	if err == nil {
		pid, err = strconv.Atoi(string(data))
	}
	return
}

func (d *Daemon) Stop() (err error) {
	if !d.IsRunning() {
		return errors.New(d.name + " does not exist")
	}

	if d.foreground {
		err = d.service.Stop()
		if err != nil {
			return
		}

		err = os.Remove(d.getPidFilename())
		if err != nil {
			return
		}

		d.running = false
		return
	}

	var process *os.Process
	process, err = os.FindProcess(d.pid)
	if err != nil {
		return
	}
	err = process.Signal(os.Interrupt)
	if err != nil {
		return
	}

	d.running = isOSProcessRunning(d.pid)
	if d.running {
		return errors.New(fmt.Sprintf("cannot stop the process:%d", d.pid))
	}

	return
}

func (d *Daemon) GetPid() int {
	return d.pid
}
