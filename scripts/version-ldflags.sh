#!/usr/bin/env bash
# Print -ldflags to stamp main.Version at link time.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$("${ROOT}/scripts/version.sh")"
printf '%s' "-X main.Version=${VERSION}"
