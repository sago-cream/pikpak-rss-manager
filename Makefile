.DEFAULT_GOAL := help
.NOTPARALLEL:

GO ?= go
GO_PACKAGES := ./cmd/... ./internal/... ./tests/...
NPM ?= npm
IMAGE ?= ghcr.io/wade00754/pikpak-rss-manager:latest
VERSION := $(strip $(file <cmd/pikpak-rss-manager/VERSION))

ifeq ($(OS),Windows_NT)
SHELL := cmd.exe
.SHELLFLAGS := /c
EXE := .exe
endif

BINARY ?= .local/pikpak-rss-manager$(EXE)

.PHONY: help dev version build check-format test test-race vet test-ui verify docker-build test-container test-manifest frontend-install frontend-build dev-frontend

help:
	@echo make dev             - run the Go API and embedded frontend
	@echo make dev-frontend    - run Vite with hot reload on port 5173
	@echo make version         - print the application version
	@echo make build           - build into .local
	@echo make check-format    - check Go formatting without rewriting files
	@echo make test            - run Go tests
	@echo make test-race       - run Go tests with the race detector
	@echo make vet             - run Go static checks
	@echo make test-ui         - check React types, lint, formatting and tests
	@echo make verify          - run formatting, tests, vet, UI checks and build
	@echo make docker-build    - build IMAGE with Docker
	@echo make test-container  - test IMAGE startup and persistence with Docker
	@echo make test-manifest   - check IMAGE has linux/amd64 and linux/arm64

dev: frontend-build
	$(GO) run ./cmd/pikpak-rss-manager

version:
	$(GO) run ./cmd/pikpak-rss-manager version

build: frontend-build
	$(GO) build -trimpath -o $(BINARY) ./cmd/pikpak-rss-manager

check-format: export FORMAT_CHECK = 1
check-format:
	$(GO) test ./tests/tooling -run ^TestGoFormatting$$ -count=1

test: frontend-build
	$(GO) test $(GO_PACKAGES)

test-race: frontend-build
	$(GO) test -race -count=1 $(GO_PACKAGES)

vet: frontend-build
	$(GO) vet $(GO_PACKAGES)

frontend/node_modules/.package-lock.json: frontend/package-lock.json frontend/package.json frontend/.npmrc
	$(NPM) --prefix frontend ci

frontend-install:
	$(NPM) --prefix frontend ci

frontend-build: frontend/node_modules/.package-lock.json
	$(NPM) --prefix frontend run build

dev-frontend: frontend/node_modules/.package-lock.json
	$(NPM) --prefix frontend run dev

test-ui: frontend/node_modules/.package-lock.json
	$(NPM) --prefix frontend run check

verify: check-format test vet test-ui build

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t "$(IMAGE)" .

test-container: export CONTAINER_SMOKE_TEST = 1
test-container: export SMOKE_IMAGE = $(IMAGE)
test-container:
	$(GO) test ./tests/container -run ^TestComposeSmoke$$ -count=1 -v -timeout=10m

test-manifest: export CONTAINER_MANIFEST_TEST = 1
test-manifest: export SMOKE_IMAGE = $(IMAGE)
test-manifest:
	$(GO) test ./tests/container -run ^TestPublishedArchitectures$$ -count=1 -v
