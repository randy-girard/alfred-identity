package main

import "strings"

// Version is set at link time: dev-<git-short> via scripts/build.sh / scripts/dev.sh,
// or release semver via CI (-ldflags "-X main.Version=x.y.z"). Defaults to "dev".
var Version = "dev"

func isDevVersion(v string) bool {
	return strings.HasPrefix(v, "dev")
}
