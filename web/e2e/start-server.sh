#!/usr/bin/env bash
# Starts the real Marionette binary for the Playwright suite (block 0041):
# a fresh, empty configuration in a temporary directory on a loopback port.
set -euo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
binary="$root/bin/marionette"
if [[ ! -x "$binary" ]]; then
	echo "e2e: $binary not found; run 'make build' first" >&2
	exit 1
fi

data=$(mktemp -d)
trap 'rm -rf "$data"' EXIT
export MARIONETTE_CONFIG="$data/marionette.json"
export MARIONETTE_ADDR="127.0.0.1:${E2E_PORT:-18080}"
export MARIONETTE_LOG_LEVEL=warn
"$binary" &
server=$!
trap 'kill "$server" 2>/dev/null; wait "$server" 2>/dev/null; rm -rf "$data"' EXIT INT TERM
wait "$server"
