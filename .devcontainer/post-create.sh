#!/usr/bin/env bash
set -euo pipefail

echo "==> Downloading Go module dependencies"
go mod download

echo "==> Installing UI dependencies"
if [ -d "web" ]; then
	(cd web && npm install)
	echo "==> Installing Chromium for the end-to-end tests (make e2e)"
	(cd web && npx playwright install --with-deps chromium)
fi

echo "==> Post-create setup complete"
