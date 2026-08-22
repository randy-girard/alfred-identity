#!/usr/bin/env bash
# Print -ldflags to stamp app.Version at link time.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$("${ROOT}/scripts/version.sh")"
VERSION_PKG="github.com/alfred-identity/app/internal/app.Version"
printf '%s' "-X ${VERSION_PKG}=${VERSION}"
