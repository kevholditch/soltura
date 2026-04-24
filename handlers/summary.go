package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"soltura/llm"
	"soltura/prompts"
	"soltura/store"
)

type SummaryHandler struct {
	store  store.Store
	client llm.Completer
}

const summaryMaxTokens = 260

func NewSummaryHandler(s store.Store, c llm.Completer) *SummaryHandler {
	return &SummaryHandler{store: s, client: c}
}

// Get handles GET /api/sessions/{sessionID}/summary
func (s *SummaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	session, err := s.store.GetSession(sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	turns, err := s.store.GetTurns(sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	corrections, err := s.store.GetCorrections(sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Calculate duration
	var endTime time.Time
	if session.EndedAt != nil {
		endTime = *session.EndedAt
	} else {
		endTime = time.Now()
	}
	durationMins := int(endTime.Sub(session.StartedAt).Minutes())
	duration := fmt.Sprintf("%d minutes", durationMins)

	// Build corrections JSON
	correctionsBytes, err := json.Marshal(corrections)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	correctionsJSON := string(correctionsBytes)

	summaryPrompt := prompts.SessionSummary(session.Topic, duration, len(turns), correctionsJSON)
	sumCtx := llm.WithModelProfile(r.Context(), llm.ModelProfileFast)
	sumCtx = llm.WithMaxTokens(sumCtx, summaryMaxTokens)
	sumCtx = llm.WithPurpose(sumCtx, llm.PurposeSessionSummary)
	summary, err := s.client.Complete(sumCtx, "", []llm.Message{{Role: "user", Content: summaryPrompt}})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary":          summary,
		"correction_count": len(corrections),
		"turn_count":       len(turns),
	})
}
