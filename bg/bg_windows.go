// +build windows

package bg

import "os"

func isRunning(pid int) (running bool) {
	_, err := os.FindProcess(pid)
	return err == nil
}

func (p bgProcess) Stop() error {
	process, err := os.FindProcess(p.pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
