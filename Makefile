OLLAMA_MODEL := gemma3:12b

# Capture the backend name when passed as a second word, e.g. "make run gemma"
BACKEND := $(filter anthropic gemma,$(MAKECMDGOALS))

.PHONY: install-model run anthropic gemma build

# No-op targets so Make doesn't error when "anthropic" or "gemma" appear on the command line
anthropic gemma:
	@:

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
	@echo "Done. Run: make run gemma"

## run [anthropic|gemma]: start the app with the chosen backend
##   make run anthropic   — use the Anthropic API (requires ANTHROPIC_API_KEY)
##   make run gemma       — use the local Ollama model
run:
	@if [ "$(BACKEND)" = "gemma" ]; then \
		LLM_BACKEND=ollama go run .; \
	elif [ "$(BACKEND)" = "anthropic" ]; then \
		go run .; \
	else \
		echo "Usage: make run anthropic  OR  make run gemma"; \
		exit 1; \
	fi

## build: compile the binary
build:
	go build -o soltura .
