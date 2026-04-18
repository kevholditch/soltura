OLLAMA_MODEL := gemma4:27b

.PHONY: install-model run build

## install-model: install Ollama (via Homebrew) if absent, then pull the model
install-model:
	@command -v ollama >/dev/null 2>&1 || { \
		echo "Ollama not found — installing via Homebrew..."; \
		brew install ollama; \
	}
	@echo "Pulling $(OLLAMA_MODEL) (this may take a while on first run)..."
	ollama pull $(OLLAMA_MODEL)
	@echo "Done. Start the server with: ollama serve"
	@echo "Then run the app with: LLM_BACKEND=ollama go run ."

## run-ollama: start the app using the local Ollama model
run-ollama:
	LLM_BACKEND=ollama go run .

## run: start the app using the Anthropic API (requires ANTHROPIC_API_KEY in .env)
run:
	go run .

## build: compile the binary
build:
	go build -o soltura .
