package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"soltura/llm"
	"soltura/models"
)

type drillTestStore struct {
	unlearntCalls int
	scopedCalls   int
	scopedIDs     []string
	unlearntVocab []models.VocabEntry
	scopedVocab   []models.VocabEntry
}

func (s *drillTestStore) CreateSession(string) (*models.Session, error) { return nil, nil }
func (s *drillTestStore) EndSession(string) error                       { return nil }
func (s *drillTestStore) GetSession(string) (*models.Session, error)    { return nil, nil }
func (s *drillTestStore) SetSessionSeedContent(string, string) error    { return nil }
func (s *drillTestStore) ListSessions(int) ([]models.SessionListItem, error) {
	return nil, nil
}
func (s *drillTestStore) GetSessionReview(string) (*models.SessionReview, error) {
	return nil, nil
}
func (s *drillTestStore) SaveTurn(string, string, string, []models.Correction) (*models.Turn, error) {
	return nil, nil
}
func (s *drillTestStore) GetTurns(string) ([]models.Turn, error)             { return nil, nil }
func (s *drillTestStore) GetCorrections(string) ([]models.Correction, error) { return nil, nil }
func (s *drillTestStore) UpsertVocab([]models.Correction) error              { return nil }
func (s *drillTestStore) GetVocab(int) ([]models.VocabEntry, error)          { return nil, nil }
func (s *drillTestStore) GetVocabSorted(int, string) ([]models.VocabEntry, error) {
	return nil, nil
}
func (s *drillTestStore) GetVocabByIDs(ids []string) ([]models.VocabEntry, error) {
	s.scopedCalls += 1
	s.scopedIDs = append([]string(nil), ids...)
	return s.scopedVocab, nil
}
func (s *drillTestStore) GetVocabCount() (int, error) { return 0, nil }
func (s *drillTestStore) GetUnlearntVocab(int) ([]models.VocabEntry, error) {
	s.unlearntCalls += 1
	return s.unlearntVocab, nil
}
func (s *drillTestStore) MarkVocabLearnt([]string) error { return nil }

type drillTestClient struct {
	completeCalls int
	lastMessages  []llm.Message
}

func (c *drillTestClient) Complete(ctx context.Context, system string, messages []llm.Message) (string, error) {
	c.completeCalls += 1
	c.lastMessages = append([]llm.Message(nil), messages...)
	return `{"pattern_name":"p","explanation":"e","question":"q","vocab_ids":["scoped-2"]}`, nil
}

func (c *drillTestClient) StreamCompletion(context.Context, string, []llm.Message, func(string)) (string, error) {
	return "", nil
}

func TestDrillStartUsesScopedVocabIDsWhenProvided(t *testing.T) {
	store := &drillTestStore{
		scopedVocab: []models.VocabEntry{
			{ID: "scoped-2", Original: "a", Corrected: "b", Explanation: "because", Category: "grammar", SeenCount: 1, LastSeen: time.Now()},
		},
		unlearntVocab: []models.VocabEntry{
			{ID: "unscoped-1", Original: "x", Corrected: "y", Explanation: "because", Category: "grammar", SeenCount: 1, LastSeen: time.Now()},
		},
	}
	client := &drillTestClient{}
	handler := NewDrillHandler(store, client)

	req := httptest.NewRequest(http.MethodPost, "/api/drills/start", strings.NewReader(`{"vocab_ids":["scoped-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Start(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if store.scopedCalls != 1 {
		t.Fatalf("expected GetVocabByIDs once, got %d", store.scopedCalls)
	}
	if store.unlearntCalls != 0 {
		t.Fatalf("expected GetUnlearntVocab not to be called, got %d", store.unlearntCalls)
	}
	if got := strings.Join(store.scopedIDs, ","); got != "scoped-2" {
		t.Fatalf("expected scoped IDs scoped-2, got %q", got)
	}
	if client.completeCalls != 1 {
		t.Fatalf("expected LLM start call once, got %d", client.completeCalls)
	}
	if len(client.lastMessages) != 1 || !strings.Contains(client.lastMessages[0].Content, "scoped-2") {
		t.Fatalf("expected prompt to include scoped vocab, got %#v", client.lastMessages)
	}
	if strings.Contains(client.lastMessages[0].Content, "unscoped-1") {
		t.Fatalf("expected prompt not to include unscoped vocab, got %s", client.lastMessages[0].Content)
	}
}

func TestDrillStartScopedEmptyResultReturnsAllDone(t *testing.T) {
	store := &drillTestStore{}
	client := &drillTestClient{}
	handler := NewDrillHandler(store, client)

	req := httptest.NewRequest(http.MethodPost, "/api/drills/start", strings.NewReader(`{"vocab_ids":["missing"]}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Start(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if store.scopedCalls != 1 {
		t.Fatalf("expected GetVocabByIDs once, got %d", store.scopedCalls)
	}
	if store.unlearntCalls != 0 {
		t.Fatalf("expected GetUnlearntVocab not to be called, got %d", store.unlearntCalls)
	}
	if client.completeCalls != 0 {
		t.Fatalf("expected LLM not to be called, got %d", client.completeCalls)
	}

	var body map[string]bool
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body["all_done"] {
		t.Fatalf("expected all_done true, got %#v", body)
	}
}

func TestDrillStartScopedEmptyIDsReturnAllDone(t *testing.T) {
	store := &drillTestStore{
		unlearntVocab: []models.VocabEntry{
			{ID: "unscoped-1", Original: "x", Corrected: "y", Explanation: "because", Category: "grammar", SeenCount: 1, LastSeen: time.Now()},
		},
	}
	client := &drillTestClient{}
	handler := NewDrillHandler(store, client)

	req := httptest.NewRequest(http.MethodPost, "/api/drills/start", strings.NewReader(`{"vocab_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Start(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if store.scopedCalls != 1 {
		t.Fatalf("expected GetVocabByIDs once, got %d", store.scopedCalls)
	}
	if store.unlearntCalls != 0 {
		t.Fatalf("expected GetUnlearntVocab not to be called, got %d", store.unlearntCalls)
	}
	if client.completeCalls != 0 {
		t.Fatalf("expected LLM not to be called, got %d", client.completeCalls)
	}

	var body map[string]bool
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body["all_done"] {
		t.Fatalf("expected all_done true, got %#v", body)
	}
}
