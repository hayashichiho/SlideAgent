.PHONY: setup dev server client api worker demo clean check-go

GO ?= go
GOCACHE_DIR ?= $(CURDIR)/.cache/go-build

check-go:
	@command -v go >/dev/null 2>&1 || { \
		echo "Error: Go is not installed or not in PATH."; \
		echo "Install Go from https://go.dev/dl/ and restart your shell."; \
		exit 1; \
	}

setup: check-go
	cd pptx-engine && npm i
	cd mcp-server && GOCACHE=$(GOCACHE_DIR) $(GO) mod tidy
	cd client && GOCACHE=$(GOCACHE_DIR) $(GO) mod tidy
	cd services/api && GOCACHE=$(GOCACHE_DIR) $(GO) mod tidy
	cd services/worker && GOCACHE=$(GOCACHE_DIR) $(GO) mod tidy

server: check-go
	cd mcp-server && GOCACHE=$(GOCACHE_DIR) $(GO) run .

client: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd client && GOCACHE=$(GOCACHE_DIR) $(GO) run .

api: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd services/api && GOCACHE=$(GOCACHE_DIR) $(GO) run .

worker: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd services/worker && GOCACHE=$(GOCACHE_DIR) $(GO) run .

dev: client

demo:
	@echo "CLI: make client"
	@echo "Web phase(min): Terminal A = make api"
	@echo "                Terminal B = make worker"
	@echo "                POST /jobs then GET /jobs/:id"

clean:
	rm -rf output/*
