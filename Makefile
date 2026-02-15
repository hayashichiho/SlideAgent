.PHONY: setup dev server client api worker frontend profile check coach demo clean check-go

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
	cd services/frontend && GOCACHE=$(GOCACHE_DIR) $(GO) mod tidy

server: check-go
	cd mcp-server && GOCACHE=$(GOCACHE_DIR) $(GO) run .

client: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd client && GOCACHE=$(GOCACHE_DIR) $(GO) run .

api: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd services/api && GOCACHE=$(GOCACHE_DIR) $(GO) run .

worker: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd services/worker && GOCACHE=$(GOCACHE_DIR) $(GO) run .

frontend: check-go
	set -a; [ -f .env ] && . ./.env; set +a; cd services/frontend && GOCACHE=$(GOCACHE_DIR) $(GO) run .

profile:
	@test -f assets/template.pptx || (echo "missing assets/template.pptx"; exit 1)
	@mkdir -p outputs
	node pptx-engine/src/pptx_extract_style.mjs --pptx assets/template.pptx --out outputs/template_metrics.json
	node pptx-engine/src/profile_build.mjs --template outputs/template_metrics.json --rules-file assets/rules_text.txt --out outputs/profile.json
	@echo "profile generated: outputs/profile.json"

check:
	@test -n "$(DECK)" || (echo "Usage: make check DECK=path/to/user.pptx"; exit 1)
	@test -f outputs/profile.json || (echo "missing outputs/profile.json. run: make profile"; exit 1)
	@mkdir -p outputs
	node pptx-engine/src/pptx_extract_style.mjs --pptx "$(DECK)" --out outputs/deck_metrics.json
	node pptx-engine/src/profile_check.mjs --profile outputs/profile.json --deck outputs/deck_metrics.json --out outputs/violations.json
	@echo "violations generated: outputs/violations.json"

coach:
	@test -f outputs/profile.json || (echo "missing outputs/profile.json. run: make profile"; exit 1)
	@test -f outputs/violations.json || (echo "missing outputs/violations.json. run: make check DECK=..."; exit 1)
	@test -n "$$GEMINI_API_KEY" || (echo "GEMINI_API_KEY is required"; exit 1)
	node pptx-engine/src/generate_tips.mjs --profile outputs/profile.json --violations outputs/violations.json --out outputs/tips.json
	@echo "tips generated: outputs/tips.json"

dev: client

demo:
	@echo "CLI: make client"
	@echo "Web phase(min): Terminal A = make api"
	@echo "                Terminal B = make worker"
	@echo "                Terminal C = make frontend"
	@echo "                open http://127.0.0.1:3000"
	@echo "Profile: make profile"
	@echo "Check:   make check DECK=output/your_deck.pptx"
	@echo "Coach:   GEMINI_API_KEY=... make coach"

clean:
	rm -rf output/*
