package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"soltura/anthropic"
	"soltura/handlers"
	"soltura/llm"
	"soltura/ollama"
	"soltura/store"
)

func loadEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func newLLMClient() (llm.Completer, func()) {
	backend := strings.ToLower(os.Getenv("LLM_BACKEND"))
	if backend == "" {
		backend = "anthropic"
	}

	switch backend {
	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "gemma4:27b"
		}
		server, err := ollama.EnsureServer(baseURL)
		if err != nil {
			log.Fatalf("ollama: %v", err)
		}
		if err := ollama.EnsureModel(baseURL, model); err != nil {
			// Leave the server running so the user can run `make install-model`
			log.Fatalf("ollama: %v", err)
		}
		log.Printf("LLM backend: ollama  url=%s  model=%s", baseURL, model)
		return ollama.NewClient(baseURL, model), server.Stop

	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			log.Fatal("ANTHROPIC_API_KEY environment variable is required when LLM_BACKEND=anthropic")
		}
		log.Printf("LLM backend: anthropic")
		return anthropic.NewClient(apiKey), func() {}

	default:
		log.Fatalf("unknown LLM_BACKEND %q — valid values: anthropic, ollama", backend)
		return nil, func() {}
	}
}

func main() {
	loadEnv()

	sqliteStore, err := store.NewSQLiteStore("./spanish.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	client, stopLLM := newLLMClient()

	// Graceful shutdown: stop the LLM server on SIGINT / SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		stopLLM()
		os.Exit(0)
	}()

	sessionHandler := handlers.NewSessionHandler(sqliteStore, client)
	summaryHandler := handlers.NewSummaryHandler(sqliteStore, client)
	vocabHandler := handlers.NewVocabHandler(sqliteStore)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/sessions", sessionHandler.Create)
	r.Post("/api/sessions/{sessionID}/turns", sessionHandler.Turn)
	r.Post("/api/sessions/{sessionID}/end", sessionHandler.End)
	r.Get("/api/sessions/{sessionID}/summary", summaryHandler.Get)
	r.Get("/api/vocab", vocabHandler.List)

	r.Handle("/*", http.FileServer(http.Dir("./web")))

	log.Println("Starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
