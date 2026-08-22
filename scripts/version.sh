#!/usr/bin/env bash
# Print the dev version string from git (used by build/dev scripts).
set -euo pipefail

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	echo "dev"
	exit 0
fi

hash="$(git rev-parse --short HEAD)"
version="dev-${hash}"
if ! git diff --quiet HEAD 2>/dev/null; then
	version="${version}+dirty"
fi
echo "$version"
