package builtin

import (
	"os"
	"syscall"
)

func platformSpecificSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func kill(proc *os.Process) error {
	return proc.Kill()
}
