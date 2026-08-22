#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$PATH:$(go env GOPATH)/bin"
if ! command -v wails >/dev/null; then
	go install github.com/wailsapp/wails/v2/cmd/wails@latest
fi
LDFLAGS="$("./scripts/version-ldflags.sh")"
echo "→ wails build (${LDFLAGS#-X github.com/alfred-identity/app/internal/app.Version=})"
wails build -clean -ldflags "$LDFLAGS" "$@"
echo "Built under build/bin/"
ls -la build/bin/ || true
