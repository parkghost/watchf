package daemon

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

const PidFileSuffix = ".pid"

type Service interface {
	Start() error
	Stop() error
}

type Daemon struct {
	name    string
	local   bool
	pid     int
	service Service
}

func (d *Daemon) Start() (err error) {
	if d.IsRunning() {
		log.Fatalln(d.name + " is already running")
		return
	}
	if err = ioutil.WriteFile("."+d.name+PidFileSuffix, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return
	}
	d.local = true
	return d.service.Start()
}

func (d *Daemon) Stop() (err error) {
	if d.IsRunning() {
		if d.local {
			os.Remove("." + d.name + PidFileSuffix)
			return d.service.Stop()
		}
		var process *os.Process
		process, err = os.FindProcess(d.pid)
		if err != nil {
			return
		}
		err = process.Signal(os.Interrupt)

	}
	return
}

func (d *Daemon) IsRunning() bool {
	if d.local {
		return true
	}

	var err error
	d.pid, err = d.readPidFromFile()
	if err == nil {
		return isProcessRunning(d.pid)
	}
	return false
}

func (d *Daemon) GetPid() int {
	return d.pid
}

func (d *Daemon) readPidFromFile() (pid int, err error) {
	data, err := ioutil.ReadFile(d.name + PidFileSuffix)
	if err == nil {
		pid, err = strconv.Atoi(string(data))
	}
	return
}

func NewDaemon(name string, service Service) *Daemon {
	return &Daemon{name: name, service: service}
}
