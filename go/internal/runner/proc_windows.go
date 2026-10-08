//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	// CREATE_NEW_PROCESS_GROUP so console Ctrl-C reaches us first and we decide.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
}

func interruptTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = exec.Command("taskkill", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
}

func killTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
}
