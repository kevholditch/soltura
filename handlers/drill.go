package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"soltura/llm"
	"soltura/prompts"
	"soltura/store"
)

type DrillHandler struct {
	store  store.Store
	client llm.Completer
}

func NewDrillHandler(s store.Store, c llm.Completer) *DrillHandler {
	return &DrillHandler{store: s, client: c}
}

// Start handles POST /api/drills/start
func (d *DrillHandler) Start(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vocab, err := d.store.GetUnlearntVocab(100)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if len(vocab) == 0 {
		json.NewEncoder(w).Encode(map[string]bool{"all_done": true})
		return
	}

	vocabJSON, err := json.Marshal(vocab)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	prompt := prompts.DrillStart(string(vocabJSON))
	result, err := d.client.Complete(r.Context(), "", []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	cleaned := strings.TrimSpace(result)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		if len(lines) > 2 {
			cleaned = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var parsed struct {
		PatternName string   `json:"pattern_name"`
		Explanation string   `json:"explanation"`
		Question    string   `json:"question"`
		VocabIDs    []string `json:"vocab_ids"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		log.Printf("drill start parse error: %v, raw: %s", err, result)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to parse LLM response"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"pattern_name": parsed.PatternName,
		"explanation":  parsed.Explanation,
		"question":     parsed.Question,
		"vocab_ids":    parsed.VocabIDs,
	})
}

// Turn handles POST /api/drills/turn (SSE)
func (d *DrillHandler) Turn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Answer      string        `json:"answer"`
		History     []llm.Message `json:"history"`
		PatternName string        `json:"pattern_name"`
		Explanation string        `json:"explanation"`
		Question    string        `json:"question"`
		VocabIDs    []string      `json:"vocab_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

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

	historyJSON, _ := json.Marshal(body.History)

	var mastered bool
	var correct bool
	var nextQuestion string
	var evalErr error

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine A: stream feedback
	go func() {
		defer wg.Done()
		system := prompts.DrillFeedback(body.PatternName, body.Question, body.Answer)
		_, streamErr := d.client.StreamCompletion(ctx, system, []llm.Message{{Role: "user", Content: body.Answer}}, func(chunk string) {
			data, _ := json.Marshal(map[string]string{"type": "chunk", "text": chunk})
			writeSSE(w, flusher, string(data))
		})
		if streamErr != nil {
			log.Printf("drill feedback stream error: %v", streamErr)
		}
	}()

	// Goroutine B: evaluate mastery
	go func() {
		defer wg.Done()
		evalPrompt := prompts.DrillEvaluate(body.PatternName, body.Explanation, body.Question, body.Answer, string(historyJSON))
		result, err := d.client.Complete(ctx, "", []llm.Message{{Role: "user", Content: evalPrompt}})
		if err != nil {
			evalErr = err
			log.Printf("drill evaluate error: %v", err)
			return
		}

		cleaned := strings.TrimSpace(result)
		if strings.HasPrefix(cleaned, "```") {
			lines := strings.Split(cleaned, "\n")
			if len(lines) > 2 {
				cleaned = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}

		var parsed struct {
			Correct      bool   `json:"correct"`
			Mastered     bool   `json:"mastered"`
			NextQuestion string `json:"next_question"`
		}
		if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
			log.Printf("drill evaluate parse error: %v, raw: %s", err, result)
			return
		}
		correct = parsed.Correct
		mastered = parsed.Mastered
		nextQuestion = parsed.NextQuestion
	}()

	wg.Wait()
	_ = evalErr

	if mastered && len(body.VocabIDs) > 0 {
		if err := d.store.MarkVocabLearnt(body.VocabIDs); err != nil {
			log.Printf("mark vocab learnt error: %v", err)
		}
	}

	// If mastered, stream a transitional celebration message before the result event
	if mastered {
		writeSSE(w, flusher, `{"type":"transition_start"}`)
		transSystem := prompts.DrillTransition(body.PatternName)
		_, transErr := d.client.StreamCompletion(ctx, transSystem, []llm.Message{{Role: "user", Content: "next"}}, func(chunk string) {
			data, _ := json.Marshal(map[string]string{"type": "transition_chunk", "text": chunk})
			writeSSE(w, flusher, string(data))
		})
		if transErr != nil {
			log.Printf("drill transition stream error: %v", transErr)
		}
	}

	resultData, _ := json.Marshal(map[string]interface{}{
		"type":          "drill_result",
		"correct":       correct,
		"mastered":      mastered,
		"next_question": nextQuestion,
	})
	writeSSE(w, flusher, string(resultData))
	writeSSE(w, flusher, `{"type":"done"}`)
}
