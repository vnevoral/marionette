SHELL := /bin/bash
APP_NAME := marionette
DIST_DIR := internal/webui/dist
DEV_CONFIG := ./marionette.json
DEV_FIXTURE := deploy/dev-fixture.json

.PHONY: ui-install ui-build ui-dev ui-lint dev-config backend-run backend-dev build build-arm64 release-arm64 test lint verify clean

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

## Create the local dev configuration from the fixture if it does not exist yet
dev-config:
	@test -f $(DEV_CONFIG) || { cp $(DEV_FIXTURE) $(DEV_CONFIG) && echo "created $(DEV_CONFIG) from $(DEV_FIXTURE)"; }

## Run the Go backend using the currently embedded UI build
backend-run: dev-config
	MARIONETTE_CONFIG=$(DEV_CONFIG) go run ./cmd/marionette

## Run the Go backend with hot reload (air)
backend-dev: dev-config
	air

## Build the UI, then produce a single Go binary for the current host platform
build: ui-build
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/marionette

## Build the UI, then cross-compile for linux/arm64
build-arm64: ui-build
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/$(APP_NAME)-linux-arm64 ./cmd/marionette

## Build and package the linux/arm64 release with systemd installation files
release-arm64: build-arm64
	tar -czf bin/$(APP_NAME)-linux-arm64.tar.gz \
		-C bin $(APP_NAME)-linux-arm64 \
		-C ../deploy marionette.service marionette.default marionette.example.json install.sh

## Run all automated tests (Go with the race detector; UI tests are added by block 0030)
test:
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
