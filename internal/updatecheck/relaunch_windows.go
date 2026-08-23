//go:build windows

package updatecheck

import (
	"fmt"
	"os/exec"
	"syscall"
)

func scheduleRelaunch(pid int, target installTarget) error {
	// Wait until the current process exits, then start the replaced binary.
	script := fmt.Sprintf(
		"while (Get-Process -Id %d -ErrorAction SilentlyContinue) { Start-Sleep -Milliseconds 200 }; Start-Process -FilePath %s",
		pid,
		powershellQuote(target.Path),
	)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func powershellQuote(s string) string {
	// Single-quoted PowerShell string; escape embedded single quotes.
	out := "'"
	for _, r := range s {
		if r == '\'' {
			out += "''"
		} else {
			out += string(r)
		}
	}
	out += "'"
	return out
}
