//go:build !windows

package builtin

import (
	"os"
	"syscall"
)

func platformSpecificSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func kill(proc *os.Process) error {
	return syscall.Kill(-proc.Pid, syscall.SIGTERM)
}
