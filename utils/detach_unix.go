//go:build !windows

package utils

import "syscall"

func detachProcess() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
