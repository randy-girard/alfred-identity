//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modKernel32          = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW     = modKernel32.NewProc("CreateMutexW")
	windowsInstanceMutex syscall.Handle
)

func acquireInstanceLock() error {
	name, err := syscall.UTF16PtrFromString("Local\\com.alfred-identity.app.instance")
	if err != nil {
		return err
	}
	r0, _, e1 := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(name)))
	handle := syscall.Handle(r0)
	if handle == 0 {
		if e1 != nil {
			return e1
		}
		return fmt.Errorf("CreateMutex failed")
	}
	const ERROR_ALREADY_EXISTS = 183
	if e1 == syscall.Errno(ERROR_ALREADY_EXISTS) {
		_ = syscall.CloseHandle(handle)
		return fmt.Errorf("already running")
	}
	windowsInstanceMutex = handle
	return nil
}

func releaseSingleInstance() {
	if windowsInstanceMutex != 0 {
		_ = syscall.CloseHandle(windowsInstanceMutex)
		windowsInstanceMutex = 0
	}
}
