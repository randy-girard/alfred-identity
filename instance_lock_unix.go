//go:build unix

package main

import (
	"os"
	"syscall"
)

var instanceLockFile *os.File

func acquireInstanceLock() error {
	path, err := instanceLockPath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return err
	}
	instanceLockFile = f
	return nil
}

func releaseSingleInstance() {
	if instanceLockFile == nil {
		return
	}
	_ = syscall.Flock(int(instanceLockFile.Fd()), syscall.LOCK_UN)
	_ = instanceLockFile.Close()
	instanceLockFile = nil
}
