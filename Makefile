# Manus AI web app — developer shortcuts.
GO ?= go
BINARY ?= bin/manusweb
PORT ?= 8080

.PHONY: help fmt vet test build run icons docker apk-check clean

help: ## Show the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-12s %s\n", $$1, $$2}'

fmt: ## Format the Go sources
	gofmt -l -w cmd internal web tools

vet: ## Run go vet
	$(GO) vet ./cmd/... ./internal/... ./web/... ./tools/...

test: ## Run the unit tests
	$(GO) test ./... -race -cover

build: ## Build the web server binary
	$(GO) build -o $(BINARY) ./cmd/manusweb

run: ## Run the server locally (requires MANUS_API_KEY or a key entered in the UI)
	PORT=$(PORT) $(GO) run ./cmd/manusweb

icons: ## Regenerate the PWA and Android launcher icons
	$(GO) run ./tools/genicons

docker: ## Build the container image
	docker build -t manus-ai-web:local .

apk-check: ## Verify the Android project layout (full build runs in CI)
	cd android && gradle assembleDebug --no-daemon

clean: ## Remove build output
	rm -rf bin dist android/app/build android/build android/.gradle
