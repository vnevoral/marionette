SHELL := /bin/bash
APP_NAME := marionette
DIST_DIR := internal/webui/dist

.PHONY: ui-install ui-build ui-dev backend-run backend-dev build build-arm64 test lint clean

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
	go run ./cmd/marionette

## Run the Go backend with hot reload (air)
backend-dev:
	air

## Build the UI, then produce a single Go binary for the current host platform
build: ui-build
	go build -o bin/$(APP_NAME) ./cmd/marionette

## Build the UI, then cross-compile for Raspberry Pi (Ubuntu, linux/arm64)
build-arm64: ui-build
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/$(APP_NAME)-linux-arm64 ./cmd/marionette

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin $(DIST_DIR)
