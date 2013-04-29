// +build windows

package main

import "os"

func isProcessRunning(pid int) (running bool) {
	_, err := os.FindProcess(pid)
	return err == nil
}
