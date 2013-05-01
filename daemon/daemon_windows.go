// +build windows

package daemon

import "os"

func isProcessRunning(pid int) (running bool) {
	_, err := os.FindProcess(pid)
	return err == nil
}
