//go:build windows

package utils

import "syscall"

const CREATE_NEW_PROCESS_GROUP = 0x00000200

func detachProcess() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: CREATE_NEW_PROCESS_GROUP,
	}
}
