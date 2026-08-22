package app

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func showAlreadyRunningError() {
	const title = AppName
	const msg = AppName + " is already running.\n\nOnly one instance can be open at a time."

	switch runtime.GOOS {
	case "darwin":
		// display alert is more reliable than display dialog for GUI launches.
		script := fmt.Sprintf(
			`display alert %q message %q as critical buttons {"OK"} default button "OK"`,
			title, msg,
		)
		cmd := exec.Command("osascript", "-e", script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, msg)
		}
	case "windows":
		ps := fmt.Sprintf(
			`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show(%q, %q, 'OK', 'Error') | Out-Null`,
			msg, title,
		)
		_ = exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	default:
		if err := exec.Command("zenity", "--error", "--title="+title, "--text="+msg).Run(); err != nil {
			_ = exec.Command("notify-send", title, msg).Run()
			fmt.Fprintln(os.Stderr, msg)
		}
	}
}
