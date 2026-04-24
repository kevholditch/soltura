package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"soltura/llm"
	"soltura/models"
	"soltura/prompts"
	"soltura/store"
)

type SessionHandler struct {
	store  store.Store
	client llm.Completer
}

const (
	sessionSeedMaxTokens       = 80
	sessionCorrectionMaxTokens = 1000
)

func NewSessionHandler(s store.Store, c llm.Completer) *SessionHandler {
	return &SessionHandler{store: s, client: c}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// Create handles POST /api/sessions
func (s *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Topic string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	session, err := s.store.CreateSession(body.Topic)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	prompt := "Ask a single short, natural opening question in Spanish to start a conversation about: " + body.Topic + ". One sentence only. No preamble, no introduction, no explanation."
	seedCtx := llm.WithModelProfile(r.Context(), llm.ModelProfileFast)
	seedCtx = llm.WithMaxTokens(seedCtx, sessionSeedMaxTokens)
	seedCtx = llm.WithPurpose(seedCtx, llm.PurposeSessionSeed)
	seedContent, err := s.client.Complete(seedCtx, "", []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id":   session.ID,
		"seed_content": seedContent,
	})
}

// Turn handles POST /api/sessions/{sessionID}/turns
func (s *SessionHandler) Turn(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var body struct {
		UserText string        `json:"user_text"`
		History  []llm.Message `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	userText := body.UserText
	history := body.History

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	session, err := s.store.GetSession(sessionID)
	if err != nil {
		errData, _ := json.Marshal(map[string]string{"type": "error", "error": err.Error()})
		writeSSE(w, flusher, string(errData))
		return
	}

	// Limit history to last 40 messages
	if len(history) > 40 {
		history = history[len(history)-40:]
	}

	// Append user message to history for conversation call
	msgs := append(history, llm.Message{Role: "user", Content: userText})

	var fullReply string
	var corrections []models.Correction
	var corrErr error

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine A: streaming conversation reply
	go func() {
		defer wg.Done()
		system := prompts.ConversationSystem(session.Topic)
		var streamErr error
		convCtx := llm.WithModelProfile(ctx, llm.ModelProfileStrong)
		convCtx = llm.WithPurpose(convCtx, llm.PurposeConversationStream)
		fullReply, streamErr = s.client.StreamCompletion(convCtx, system, msgs, func(chunk string) {
			data, _ := json.Marshal(map[string]string{"type": "chunk", "text": chunk})
			writeSSE(w, flusher, string(data))
		})
		if streamErr != nil {
			log.Printf("stream error: %v", streamErr)
		}
	}()

	// Goroutine B: correction analysis
	go func() {
		defer wg.Done()
		corrPrompt := prompts.CorrectionAnalysis(userText)
		corrMsgs := []llm.Message{{Role: "user", Content: corrPrompt}}
		corrCtx := llm.WithModelProfile(ctx, llm.ModelProfileFast)
		corrCtx = llm.WithMaxTokens(corrCtx, sessionCorrectionMaxTokens)
		corrCtx = llm.WithPurpose(corrCtx, llm.PurposeCorrectionAnalysis)
		result, err := s.client.Complete(corrCtx, "", corrMsgs)
		if err != nil {
			corrErr = err
			log.Printf("correction error: %v", err)
			return
		}

		parsedCorrections, err := parseCorrectionsPayload(result)
		if err != nil {
			log.Printf("parse corrections error: %v, raw: %s", err, result)
			return
		}
		corrections = parsedCorrections
	}()

	wg.Wait()

	// Suppress unused variable warning
	_ = corrErr

	// Save turn and vocab
	_, err = s.store.SaveTurn(sessionID, userText, fullReply, corrections)
	if err != nil {
		log.Printf("save turn error: %v", err)
	}
	if len(corrections) > 0 {
		s.store.UpsertVocab(corrections)
	}

	// Ensure corrections is not nil for JSON serialisation
	if corrections == nil {
		corrections = []models.Correction{}
	}

	corrData, _ := json.Marshal(map[string]interface{}{"type": "corrections", "corrections": corrections})
	writeSSE(w, flusher, string(corrData))

	writeSSE(w, flusher, `{"type":"done"}`)
}

// End handles POST /api/sessions/{sessionID}/end
func (s *SessionHandler) End(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if err := s.store.EndSession(sessionID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
