#!/usr/bin/env bash
set -euo pipefail

echo "==> Downloading Go module dependencies"
go mod download

echo "==> Installing UI dependencies"
if [ -d "web" ]; then
	(cd web && npm install)
fi

echo "==> Post-create setup complete"
