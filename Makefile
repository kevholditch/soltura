OLLAMA_MODEL := gemma3:12b
BACKEND      := $(filter anthropic,$(MAKECMDGOALS))

.PHONY: install-model run anthropic build dev build-frontend install-playwright test-go test-e2e test-all

# No-op target so Make doesn't error when "anthropic" appears on the command line
anthropic:
	@:

frontend/node_modules:
	cd frontend && npm install

## build-frontend: compile React source into web/
build-frontend: frontend/node_modules
	cd frontend && npm run build

## build: compile React + Go binary
build: build-frontend
	go build -o soltura .

## run [anthropic]: build frontend then start Go server (default: local Gemma)
##   make run             — use local Gemma model
##   make run anthropic   — use Anthropic API (requires ANTHROPIC_API_KEY)
run: build-frontend
	@if [ "$(BACKEND)" = "anthropic" ]; then \
		go run .; \
	else \
		LLM_BACKEND=ollama go run .; \
	fi

## dev [anthropic]: Vite hot-reload on :5173 + Go on :8080 (default: local Gemma)
##   make dev             — use local Gemma model
##   make dev anthropic   — use Anthropic API (requires ANTHROPIC_API_KEY)
dev: frontend/node_modules
	@if [ "$(BACKEND)" = "anthropic" ]; then \
		(trap 'kill %1 2>/dev/null' EXIT; (cd frontend && npm run dev) & go run .); \
	else \
		(trap 'kill %1 2>/dev/null' EXIT; (cd frontend && npm run dev) & LLM_BACKEND=ollama go run .); \
	fi

## install-model: install Ollama (via Homebrew) if absent, start server, then pull the model
install-model:
	@command -v ollama >/dev/null 2>&1 || { \
		echo "Ollama not found — installing via Homebrew..."; \
		brew install ollama; \
	}
	@curl -s http://localhost:11434/ >/dev/null 2>&1 || { \
		echo "Starting Ollama server..."; \
		ollama serve >/dev/null 2>&1 & \
		sleep 2; \
	}
	@echo "Pulling $(OLLAMA_MODEL) (this may take a while on first run)..."
	ollama pull $(OLLAMA_MODEL)
	@echo "Done. Run: make dev"

## test-go: run Go unit tests
test-go:
	go test ./...

## install-playwright: install frontend deps + Playwright browser/runtime
install-playwright:
	cd frontend && npm ci
	cd frontend && npx playwright install --with-deps chromium

## test-e2e: run Playwright end-to-end tests
test-e2e: frontend/node_modules
	cd frontend && npm run test:e2e

## test-all: run all test suites
test-all: test-go test-e2e
