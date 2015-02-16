package bg

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
)

var (
	stop       bool
	background bool

	out    string
	errOut string
)

func init() {
	flag.BoolVar(&stop, "s", false, "Stop the background process")
	flag.BoolVar(&background, "d", false, "Run as background process")

	flag.StringVar(&out, "out", ".out", "Redirect stdout to given file")
	flag.StringVar(&errOut, "err", ".out", "Redirect stderr to given file")
}

type State int

const (
	New State = iota
	Running
)

type Service interface {
	Start() error
	Stop() error
}

func Init(name string, s Service) (Service, error) {
	pidFile := pidFile(name)

	state := New
	pid, err := pid(pidFile)
	if err == nil {
		if isRunning(pid) {
			state = Running
		} else {
			state = New

			// clean borken pid file
			err = cleanPidFile(pidFile)
			if err != nil {
				return nil, err
			}
		}
	}

	switch state {
	case New:
		if stop {
			return nil, fmt.Errorf("%s has stopped", name)
		}

		fgp := fgProcess{s, name, pid}
		if background {
			args := os.Args[:0]
			for _, e := range os.Args {
				if e != "-d" {
					args = append(args, e)
				}
			}

			var f1, f2 *os.File
			f1, err = os.Create(out)
			if err != nil {
				fmt.Fprintf(os.Stderr, "err: %s", err)
			}
			if out == errOut {
				f2 = f1
			} else {
				f2, err = os.Create(errOut)
				if err != nil {
					fmt.Fprintf(os.Stderr, "err: %s", err)
				}
			}

			var cmd *exec.Cmd
			if len(args) == 1 {
				cmd = exec.Command(args[0])
			} else {
				cmd = exec.Command(args[0], args[1:]...)
			}
			cmd.Stdout = f1
			cmd.Stderr = f2
			fmt.Println("Starting background process")
			err = cmd.Start()
			if err != nil {
				fmt.Printf("Start background process failed: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("Started background process(%d)\n", cmd.Process.Pid)
			os.Exit(0)
		}

		return fgp, nil
	case Running:
		if background {
			return nil, fmt.Errorf("%s(%d) is already running", name, pid)
		}

		bgp := bgProcess{s, name, pid}
		if stop {
			err = bgp.Stop()
			if err != nil {
				return nil, fmt.Errorf("stop background process(%d) failed: %s", pid, err)
			}

			// TODO: user defined logger
			fmt.Printf("Stopped background process(%d)\n", pid)
			os.Exit(0)
		}
		return bgp, nil
	}

	panic(fmt.Sprintf("unknow %s(%d) state", name, pid))
}

func pidFile(name string) string {
	return "." + name + ".pid"
}

func pid(file string) (int, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(b))
}

func cleanPidFile(file string) error {
	err := os.Remove(file)
	if err != nil {
		return fmt.Errorf("cleanup failed: %s", err)
	}
	return nil
}

func createPidFile(file string) error {
	err := ioutil.WriteFile(file, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		return fmt.Errorf("write Pid failed: %s", err)
	}
	return nil
}

type fgProcess struct {
	service Service
	name    string
	pid     int
}

func (p fgProcess) Start() error {
	err := createPidFile(pidFile(p.name))
	if err != nil {
		return err
	}
	return p.service.Start()
}

func (p fgProcess) Stop() error {
	err := cleanPidFile(pidFile(p.name))
	if err != nil {
		return err
	}
	return p.service.Stop()
}

type bgProcess struct {
	service Service
	name    string
	pid     int
}

func (p bgProcess) Start() error {
	return fmt.Errorf("%s(%d) is already running", p.name, p.pid)
}
