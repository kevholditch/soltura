package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"soltura/models"
	"soltura/store"
)

type VocabHandler struct {
	store store.Store
}

func NewVocabHandler(s store.Store) *VocabHandler {
	return &VocabHandler{store: s}
}

// List handles GET /api/vocab
func (v *VocabHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	entries, err := v.store.GetVocab(limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	total, err := v.store.GetVocabCount()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if entries == nil {
		entries = []models.VocabEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"total":   total,
	})
}
