// +build windows

package daemon

import "os"

func isOSProcessRunning(pid int) (running bool) {
	_, err := os.FindProcess(pid)
	return err == nil
}
