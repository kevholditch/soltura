package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"soltura/llm"
	"soltura/prompts"
	"soltura/store"
)

var loadingPhrases = []string{
	"Warming up the conjugation engine…",
	"Consulting the Real Academia Española…",
	"Herding rogue accent marks…",
	"Calibrating your ser/estar detector…",
	"Untangling your prepositions…",
	"Reviewing the evidence against your grammar…",
	"Sharpening the correction pencil…",
	"Waking up the drill sergeant…",
	"Building your personalised obstacle course…",
	"Convincing the subjunctive to cooperate…",
	"Sorting mistakes by severity (spoiler: all fixable)…",
	"Polishing your irregular verbs…",
	"Checking under every tilde…",
	"Negotiating with your gender agreements…",
	"Loading the catapult with practice questions…",
}

const (
	drillStartVocabLimit  = 30
	drillHistoryMaxTurns  = 8
	drillStartMaxTokens   = 320
	drillMarkMaxTokens    = 40
	drillFeedbackMaxToken = 180
	drillEvalMaxTokens    = 220
	drillTransitionMaxTok = 90
)

type DrillHandler struct {
	store  store.Store
	client llm.Completer
}

func NewDrillHandler(s store.Store, c llm.Completer) *DrillHandler {
	return &DrillHandler{store: s, client: c}
}

// Phrases handles GET /api/drills/phrases
func (d *DrillHandler) Phrases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loadingPhrases)
}

// Start handles POST /api/drills/start
func (d *DrillHandler) Start(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vocab, err := d.store.GetUnlearntVocab(drillStartVocabLimit)
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
	startCtx := llm.WithModelProfile(r.Context(), llm.ModelProfileFast)
	startCtx = llm.WithMaxTokens(startCtx, drillStartMaxTokens)
	startCtx = llm.WithPurpose(startCtx, llm.PurposeDrillStart)
	result, err := d.client.Complete(startCtx, "", []llm.Message{{Role: "user", Content: prompt}})
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

	history := body.History
	if len(history) > drillHistoryMaxTurns {
		history = history[len(history)-drillHistoryMaxTurns:]
	}
	historyJSON, _ := json.Marshal(history)

	var correct bool
	var mastered bool
	var nextQuestion string
	// Step 1: quick lightweight correctness mark
	markPrompt := prompts.DrillMark(body.PatternName, body.Question, body.Answer)
	markCtx := llm.WithModelProfile(ctx, llm.ModelProfileFast)
	markCtx = llm.WithMaxTokens(markCtx, drillMarkMaxTokens)
	markCtx = llm.WithPurpose(markCtx, llm.PurposeDrillMark)
	markResult, err := d.client.Complete(markCtx, "", []llm.Message{{Role: "user", Content: markPrompt}})
	if err != nil {
		log.Printf("drill mark error: %v", err)
	}

	if err == nil {
		cleaned := strings.TrimSpace(markResult)
		if strings.HasPrefix(cleaned, "```") {
			lines := strings.Split(cleaned, "\n")
			if len(lines) > 2 {
				cleaned = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}
		var parsed struct {
			Correct bool `json:"correct"`
		}
		if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
			log.Printf("drill mark parse error: %v, raw: %s", err, markResult)
		} else {
			correct = parsed.Correct
		}
	}

	markData, _ := json.Marshal(map[string]interface{}{
		"type":    "mark",
		"correct": correct,
	})
	writeSSE(w, flusher, string(markData))

	// Step 2: stream coaching feedback aligned with correctness
	system := prompts.DrillFeedback(body.PatternName, body.Question, body.Answer, correct)
	feedbackCtx := llm.WithModelProfile(ctx, llm.ModelProfileFast)
	feedbackCtx = llm.WithMaxTokens(feedbackCtx, drillFeedbackMaxToken)
	feedbackCtx = llm.WithPurpose(feedbackCtx, llm.PurposeDrillFeedback)
	_, streamErr := d.client.StreamCompletion(feedbackCtx, system, []llm.Message{{Role: "user", Content: body.Answer}}, func(chunk string) {
		data, _ := json.Marshal(map[string]string{"type": "chunk", "text": chunk})
		writeSSE(w, flusher, string(data))
	})
	if streamErr != nil {
		log.Printf("drill feedback stream error: %v", streamErr)
	}

	// Step 3: decide mastery and next question
	evalPrompt := prompts.DrillEvaluate(body.PatternName, body.Explanation, body.Question, body.Answer, string(historyJSON))
	evalCtx := llm.WithModelProfile(ctx, llm.ModelProfileFast)
	evalCtx = llm.WithMaxTokens(evalCtx, drillEvalMaxTokens)
	evalCtx = llm.WithPurpose(evalCtx, llm.PurposeDrillEvaluate)
	evalResult, err := d.client.Complete(evalCtx, "", []llm.Message{{Role: "user", Content: evalPrompt}})
	if err != nil {
		log.Printf("drill evaluate error: %v", err)
	}

	if err == nil {
		cleaned := strings.TrimSpace(evalResult)
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
			log.Printf("drill evaluate parse error: %v, raw: %s", err, evalResult)
		} else {
			mastered = parsed.Mastered
			nextQuestion = parsed.NextQuestion
		}
	}

	if mastered && len(body.VocabIDs) > 0 {
		if err := d.store.MarkVocabLearnt(body.VocabIDs); err != nil {
			log.Printf("mark vocab learnt error: %v", err)
		}
	}

	if !mastered && strings.TrimSpace(nextQuestion) == "" {
		nextQuestion = body.Question
	}

	// If mastered, stream a transitional celebration message before the result event
	if mastered {
		writeSSE(w, flusher, `{"type":"transition_start"}`)
		transSystem := prompts.DrillTransition(body.PatternName)
		transCtx := llm.WithModelProfile(ctx, llm.ModelProfileFast)
		transCtx = llm.WithMaxTokens(transCtx, drillTransitionMaxTok)
		transCtx = llm.WithPurpose(transCtx, llm.PurposeDrillTransition)
		_, transErr := d.client.StreamCompletion(transCtx, transSystem, []llm.Message{{Role: "user", Content: "next"}}, func(chunk string) {
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
