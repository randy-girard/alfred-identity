//go:build unix

package updatecheck

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
)

func scheduleRelaunch(pid int, target installTarget) error {
	var launch string
	if target.Kind == "app" {
		launch = "/usr/bin/open -n " + strconv.Quote(target.Path)
	} else {
		launch = strconv.Quote(target.Path)
	}
	script := fmt.Sprintf("while kill -0 %d 2>/dev/null; do sleep 0.2; done; %s", pid, launch)
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
