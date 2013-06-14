// +build linux freebsd openbsd netbsd darwin

package daemon

import "syscall"

func isOSProcessRunning(pid int) (running bool) {
	err := syscall.Kill(pid, 0)
	return err == nil
}
