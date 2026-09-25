SHELL := /bin/bash
APP_NAME := marionette
DIST_DIR := internal/webui/dist

.PHONY: ui-install ui-build ui-dev backend-run backend-dev build build-arm64 release-arm64 test lint clean

## Install UI dependencies
ui-install:
	cd web && npm install

## Build the SPA into internal/webui/dist so it can be embedded by Go
ui-build:
	cd web && npm run build

## Run the Vite dev server (proxies /api to the Go backend on :8080)
ui-dev:
	cd web && npm run dev

## Run the Go backend using the currently embedded UI build
backend-run:
	MARIONETTE_CONFIG=./marionette.json go run ./cmd/marionette

## Run the Go backend with hot reload (air)
backend-dev:
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

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin $(DIST_DIR)
