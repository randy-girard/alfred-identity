#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$PATH:$(go env GOPATH)/bin"
if ! command -v wails >/dev/null; then
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
fi
wails build -clean
echo "Built under build/bin/"
ls -la build/bin/ || true
