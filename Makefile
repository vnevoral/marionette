SHELL := /bin/bash
APP_NAME := marionette
DIST_DIR := internal/webui/dist
DEV_CONFIG := ./marionette.json
DEV_FIXTURE := deploy/dev-fixture.json
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: ui-install ui-build ui-dev ui-lint ui-test deploy-test e2e dev-config backend-run backend-dev build build-arm64 release-arm64 test lint verify clean

## Install UI dependencies
ui-install:
	cd web && npm install

## Build the SPA into internal/webui/dist so it can be embedded by Go
ui-build:
	cd web && npm run build

## Run the Vite dev server (proxies /api to the Go backend on :8080)
ui-dev:
	cd web && npm run dev

## Lint, type-check and format-check the UI
ui-lint:
	cd web && npm run lint && npm run lint:types && npm run format:check

## Run the UI unit tests (Vitest)
ui-test:
	cd web && npm test

## Create the local dev configuration from the fixture if it does not exist yet
dev-config:
	@test -f $(DEV_CONFIG) || { cp $(DEV_FIXTURE) $(DEV_CONFIG) && echo "created $(DEV_CONFIG) from $(DEV_FIXTURE)"; }

## Run the Go backend using the currently embedded UI build
backend-run: dev-config
	MARIONETTE_CONFIG=$(DEV_CONFIG) go run -ldflags "$(LDFLAGS)" ./cmd/marionette

## Run the Go backend with hot reload (air)
backend-dev: dev-config
	air

## Build the UI, then produce a single Go binary for the current host platform
build: ui-build
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/marionette

## Build the UI, then cross-compile for linux/arm64
build-arm64: ui-build
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-linux-arm64 ./cmd/marionette

## Build and package the linux/arm64 release with systemd installation files
release-arm64: build-arm64
	tar -czf bin/$(APP_NAME)-linux-arm64.tar.gz \
		-C bin $(APP_NAME)-linux-arm64 \
		-C ../deploy marionette.service marionette.default marionette.example.json install.sh

## Staged test of the installer against the systemd unit (no root, no systemd)
deploy-test:
	bash -n deploy/install.sh
	bash deploy/install_test.sh

## End-to-end tests: real binary with the embedded SPA in headless Chromium
## (Playwright). Needs `npx playwright install chromium` once; not part of
## `verify`, runs as its own CI job.
e2e: build
	cd web && npx playwright test

## Run all automated tests (UI unit tests, installer, then Go with the race detector)
test: ui-test deploy-test
	go test -race -count=1 ./...

## Run all linters and format checks (Go + UI)
lint:
	golangci-lint run ./...
	$(MAKE) ui-lint

## Single validation entry point used by the Definition of Done and CI
verify: ui-build lint test
	go build ./...
	go vet ./...

clean:
	rm -rf bin $(DIST_DIR)
