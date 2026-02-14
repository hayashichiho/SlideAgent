.PHONY: setup dev server client demo clean check-go

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

server: check-go
	cd mcp-server && GOCACHE=$(GOCACHE_DIR) $(GO) run .

client: check-go
	cd client && GOCACHE=$(GOCACHE_DIR) $(GO) run .

dev: server

demo:
	@echo "1) Terminal A: make server"
	@echo "2) Terminal B: make client"
	@echo "3) Paste a pitch description -> get outputs/*.pptx"

clean:
	rm -rf outputs/*
