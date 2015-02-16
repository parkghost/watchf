// +build linux freebsd openbsd netbsd darwin

package bg

import (
	"os"
	"syscall"
)

func isRunning(pid int) (running bool) {
	err := syscall.Kill(pid, 0)
	return err == nil
}

func (p bgProcess) Stop() error {
	process, err := os.FindProcess(p.pid)
	if err != nil {
		return err
	}
	err = process.Signal(os.Interrupt)
	if err != nil {
		return err
	}
	return nil
}
