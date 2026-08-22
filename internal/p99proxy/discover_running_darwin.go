//go:build darwin

package p99proxy

import (
	"os/exec"
	"strings"
)

func discoverRunningDataDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			return
		}
		if hasConfig(dir) {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	for _, proc := range []string{"P99LoginProxy", "p99-login-proxy", "P99 Login Proxy"} {
		out, err := exec.Command("lsof", "-a", "-d", "cwd", "-c", proc, "-Fn").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "n") {
				add(strings.TrimPrefix(line, "n"))
			}
		}
	}
	return dirs
}
