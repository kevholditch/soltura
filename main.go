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
	"soltura/testllm"
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
			model = "gemma3:12b"
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
		strongModel, fastModel := anthropic.ResolveModels(
			os.Getenv("ANTHROPIC_MODEL_STRONG"),
			os.Getenv("ANTHROPIC_MODEL_FAST"),
		)
		log.Printf("LLM backend: anthropic  strong_model=%s  fast_model=%s", strongModel, fastModel)
		return anthropic.NewClient(apiKey, strongModel, fastModel), func() {}

	case "test":
		client, err := testllm.NewClientFromEnv()
		if err != nil {
			log.Fatalf("test backend: %v", err)
		}
		log.Printf("LLM backend: test  fixture=%s", os.Getenv("TEST_FIXTURE_PATH"))
		return client, func() {}

	default:
		log.Fatalf("unknown LLM_BACKEND %q — valid values: anthropic, ollama, test", backend)
		return nil, func() {}
	}
}

func main() {
	loadEnv()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./spanish.db"
	}

	sqliteStore, err := store.NewSQLiteStore(dbPath)
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
	drillHandler := handlers.NewDrillHandler(sqliteStore, client)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/sessions", sessionHandler.Create)
	r.Post("/api/sessions/{sessionID}/turns", sessionHandler.Turn)
	r.Post("/api/sessions/{sessionID}/end", sessionHandler.End)
	r.Get("/api/sessions/{sessionID}/summary", summaryHandler.Get)
	r.Get("/api/vocab", vocabHandler.List)
	r.Get("/api/drills/phrases", drillHandler.Phrases)
	r.Post("/api/drills/start", drillHandler.Start)
	r.Post("/api/drills/turn", drillHandler.Turn)

	r.Handle("/*", http.FileServer(http.Dir("./web")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + strings.TrimPrefix(port, ":")

	log.Printf("Starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
