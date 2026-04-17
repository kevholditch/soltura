package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"soltura/anthropic"
	"soltura/handlers"
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

func main() {
	loadEnv()

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	sqliteStore, err := store.NewSQLiteStore("./spanish.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	anthropicClient := anthropic.NewClient(apiKey)

	sessionHandler := handlers.NewSessionHandler(sqliteStore, anthropicClient)
	summaryHandler := handlers.NewSummaryHandler(sqliteStore, anthropicClient)
	vocabHandler := handlers.NewVocabHandler(sqliteStore)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/sessions", sessionHandler.Create)
	r.Post("/api/sessions/{sessionID}/turns", sessionHandler.Turn)
	r.Post("/api/sessions/{sessionID}/end", sessionHandler.End)
	r.Get("/api/sessions/{sessionID}/summary", summaryHandler.Get)
	r.Get("/api/vocab", vocabHandler.List)

	// Serve static files from ./web directory
	r.Handle("/*", http.FileServer(http.Dir("./web")))

	log.Println("Starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
