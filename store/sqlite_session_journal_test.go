package store

import (
	"reflect"
	"testing"
	"time"

	"soltura/models"
)

func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()

	s, err := NewSQLiteStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return s
}

func TestListSessionsReturnsNewestFirstWithCountsAndCategories(t *testing.T) {
	s := newTestSQLiteStore(t)

	oldSession, err := s.CreateSession("old topic")
	if err != nil {
		t.Fatalf("CreateSession old: %v", err)
	}
	newSession, err := s.CreateSession("new topic")
	if err != nil {
		t.Fatalf("CreateSession new: %v", err)
	}

	oldStart := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	newStart := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	newEnd := time.Date(2026, 4, 2, 10, 30, 0, 0, time.UTC)
	if _, err := s.db.Exec(`UPDATE sessions SET started_at = ? WHERE id = ?`, oldStart.Format(time.RFC3339), oldSession.ID); err != nil {
		t.Fatalf("update old session: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE sessions SET started_at = ?, ended_at = ? WHERE id = ?`, newStart.Format(time.RFC3339), newEnd.Format(time.RFC3339), newSession.ID); err != nil {
		t.Fatalf("update new session: %v", err)
	}

	if _, err := s.SaveTurn(oldSession.ID, "old", "reply", nil); err != nil {
		t.Fatalf("SaveTurn old: %v", err)
	}
	if _, err := s.SaveTurn(newSession.ID, "new 1", "reply 1", []models.Correction{
		{Original: "la problema", Corrected: "el problema", Explanation: "Gender", Category: "gender"},
		{Original: "yo goed", Corrected: "yo fui", Explanation: "Grammar", Category: "grammar"},
	}); err != nil {
		t.Fatalf("SaveTurn new 1: %v", err)
	}
	if _, err := s.SaveTurn(newSession.ID, "new 2", "reply 2", []models.Correction{
		{Original: "otra problema", Corrected: "otro problema", Explanation: "Gender", Category: "gender"},
	}); err != nil {
		t.Fatalf("SaveTurn new 2: %v", err)
	}

	sessions, err := s.ListSessions(10)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].ID != newSession.ID || sessions[1].ID != oldSession.ID {
		t.Fatalf("sessions ordered as ids %q, %q; want newest first", sessions[0].ID, sessions[1].ID)
	}
	if sessions[0].TurnCount != 2 {
		t.Fatalf("new session TurnCount = %d, want 2", sessions[0].TurnCount)
	}
	if sessions[0].CorrectionCount != 3 {
		t.Fatalf("new session CorrectionCount = %d, want 3", sessions[0].CorrectionCount)
	}
	if !reflect.DeepEqual(sessions[0].Categories, []string{"gender", "grammar"}) {
		t.Fatalf("new session Categories = %#v, want first-seen unique categories", sessions[0].Categories)
	}
	if sessions[0].EndedAt == nil || !sessions[0].EndedAt.Equal(newEnd) {
		t.Fatalf("new session EndedAt = %v, want %v", sessions[0].EndedAt, newEnd)
	}
	if !reflect.DeepEqual(sessions[1].Categories, []string{}) {
		t.Fatalf("old session Categories = %#v, want empty slice", sessions[1].Categories)
	}
}

func TestListSessionsExcludesSessionsWithoutTurns(t *testing.T) {
	s := newTestSQLiteStore(t)

	emptySession, err := s.CreateSession("seed only")
	if err != nil {
		t.Fatalf("CreateSession empty: %v", err)
	}
	if err := s.EndSession(emptySession.ID); err != nil {
		t.Fatalf("EndSession empty: %v", err)
	}

	realSession, err := s.CreateSession("actual practice")
	if err != nil {
		t.Fatalf("CreateSession real: %v", err)
	}
	if _, err := s.SaveTurn(realSession.ID, "hola", "hola", nil); err != nil {
		t.Fatalf("SaveTurn real: %v", err)
	}

	sessions, err := s.ListSessions(10)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1: %#v", len(sessions), sessions)
	}
	if sessions[0].ID != realSession.ID {
		t.Fatalf("listed session ID = %q, want %q", sessions[0].ID, realSession.ID)
	}
}

func TestGetSessionReviewReturnsSessionTurnsCorrectionsAndCategories(t *testing.T) {
	s := newTestSQLiteStore(t)

	session, err := s.CreateSession("review topic")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.SetSessionSeedContent(session.ID, "opening question"); err != nil {
		t.Fatalf("SetSessionSeedContent: %v", err)
	}
	firstTurn, err := s.SaveTurn(session.ID, "first user", "first reply", []models.Correction{
		{Original: "soy cansado", Corrected: "estoy cansado", Explanation: "Use estar", Category: "grammar"},
	})
	if err != nil {
		t.Fatalf("SaveTurn first: %v", err)
	}
	secondTurn, err := s.SaveTurn(session.ID, "second user", "second reply", []models.Correction{
		{Original: "el mano", Corrected: "la mano", Explanation: "Gender", Category: "gender"},
		{Original: "muy bienisimo", Corrected: "muy bien", Explanation: "Register", Category: "register"},
	})
	if err != nil {
		t.Fatalf("SaveTurn second: %v", err)
	}

	review, err := s.GetSessionReview(session.ID)
	if err != nil {
		t.Fatalf("GetSessionReview: %v", err)
	}

	if review.Session.ID != session.ID {
		t.Fatalf("review session ID = %q, want %q", review.Session.ID, session.ID)
	}
	if review.Session.SeedContent != "opening question" {
		t.Fatalf("review session SeedContent = %q, want opening question", review.Session.SeedContent)
	}
	if len(review.Turns) != 2 {
		t.Fatalf("review turns = %d, want 2", len(review.Turns))
	}
	if review.Turns[0].ID != firstTurn.ID || review.Turns[1].ID != secondTurn.ID {
		t.Fatalf("review turns not in creation order")
	}
	if len(review.Turns[1].Corrections) != 2 {
		t.Fatalf("second turn corrections = %d, want 2", len(review.Turns[1].Corrections))
	}
	if review.CorrectionCount != 3 {
		t.Fatalf("CorrectionCount = %d, want 3", review.CorrectionCount)
	}
	if !reflect.DeepEqual(review.Categories, []string{"grammar", "gender", "register"}) {
		t.Fatalf("Categories = %#v, want first-seen unique categories", review.Categories)
	}
}

func TestGetVocabByIDsReturnsOnlyRequestedUnlearntEntries(t *testing.T) {
	s := newTestSQLiteStore(t)

	corrections := []models.Correction{
		{Original: "uno", Corrected: "una", Explanation: "Gender", Category: "gender"},
		{Original: "dos", Corrected: "dos", Explanation: "Vocabulary", Category: "vocabulary"},
	}
	if err := s.UpsertVocab(corrections); err != nil {
		t.Fatalf("UpsertVocab: %v", err)
	}
	vocab, err := s.GetUnlearntVocab(10)
	if err != nil {
		t.Fatalf("GetUnlearntVocab: %v", err)
	}
	if len(vocab) != 2 {
		t.Fatalf("vocab = %d, want 2", len(vocab))
	}
	if err := s.MarkVocabLearnt([]string{vocab[0].ID}); err != nil {
		t.Fatalf("MarkVocabLearnt: %v", err)
	}

	entries, err := s.GetVocabByIDs([]string{vocab[0].ID, vocab[1].ID})
	if err != nil {
		t.Fatalf("GetVocabByIDs: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].ID != vocab[1].ID {
		t.Fatalf("entry ID = %q, want unlearnt ID %q", entries[0].ID, vocab[1].ID)
	}
	if entries[0].Learnt {
		t.Fatalf("entry Learnt = true, want false")
	}

	empty, err := s.GetVocabByIDs(nil)
	if err != nil {
		t.Fatalf("GetVocabByIDs nil: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty result = %d entries, want 0", len(empty))
	}
}

func TestGetVocabSupportsRecentAndFrequencySortsWithLearntMetadata(t *testing.T) {
	s := newTestSQLiteStore(t)

	corrections := []models.Correction{
		{Original: "frecuente", Corrected: "frecuente", Explanation: "Seen often", Category: "vocabulary"},
		{Original: "reciente", Corrected: "reciente", Explanation: "Seen lately", Category: "grammar"},
	}
	if err := s.UpsertVocab(corrections[:1]); err != nil {
		t.Fatalf("UpsertVocab frequent first: %v", err)
	}
	if err := s.UpsertVocab(corrections[:1]); err != nil {
		t.Fatalf("UpsertVocab frequent second: %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := s.UpsertVocab(corrections[1:]); err != nil {
		t.Fatalf("UpsertVocab recent: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE vocab SET last_seen = ? WHERE original = ?`, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC).Format(time.RFC3339), "frecuente"); err != nil {
		t.Fatalf("set frequent last_seen: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE vocab SET last_seen = ? WHERE original = ?`, time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC).Format(time.RFC3339), "reciente"); err != nil {
		t.Fatalf("set recent last_seen: %v", err)
	}

	recent, err := s.GetVocabSorted(10, "recent")
	if err != nil {
		t.Fatalf("GetVocab recent: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("recent entries = %d, want 2", len(recent))
	}
	if recent[0].Original != "reciente" || recent[1].Original != "frecuente" {
		t.Fatalf("recent order = %q, %q; want reciente, frecuente", recent[0].Original, recent[1].Original)
	}

	if err := s.MarkVocabLearnt([]string{recent[0].ID}); err != nil {
		t.Fatalf("MarkVocabLearnt: %v", err)
	}
	defaultSort, err := s.GetVocabSorted(10, "")
	if err != nil {
		t.Fatalf("GetVocab default sort: %v", err)
	}
	if !defaultSort[0].Learnt || defaultSort[0].LearntAt == nil {
		t.Fatalf("default entry learnt metadata = %v, %v; want learnt with learnt_at", defaultSort[0].Learnt, defaultSort[0].LearntAt)
	}

	frequency, err := s.GetVocabSorted(10, "frequency")
	if err != nil {
		t.Fatalf("GetVocab frequency: %v", err)
	}
	if frequency[0].Original != "frecuente" || frequency[1].Original != "reciente" {
		t.Fatalf("frequency order = %q, %q; want frecuente, reciente", frequency[0].Original, frequency[1].Original)
	}
}

func TestUpsertVocabReopensLearntEntryWhenMistakeReappears(t *testing.T) {
	s := newTestSQLiteStore(t)

	correction := models.Correction{
		Original: "voy a ir a el skatepark", Corrected: "voy a ir al skatepark", Explanation: "Use al", Category: "grammar",
	}
	if err := s.UpsertVocab([]models.Correction{correction}); err != nil {
		t.Fatalf("UpsertVocab first: %v", err)
	}
	vocab, err := s.GetUnlearntVocab(10)
	if err != nil {
		t.Fatalf("GetUnlearntVocab first: %v", err)
	}
	if len(vocab) != 1 {
		t.Fatalf("initial unlearnt vocab = %d, want 1", len(vocab))
	}
	if err := s.MarkVocabLearnt([]string{vocab[0].ID}); err != nil {
		t.Fatalf("MarkVocabLearnt: %v", err)
	}
	if err := s.UpsertVocab([]models.Correction{correction}); err != nil {
		t.Fatalf("UpsertVocab repeated: %v", err)
	}

	reopened, err := s.GetUnlearntVocab(10)
	if err != nil {
		t.Fatalf("GetUnlearntVocab reopened: %v", err)
	}
	if len(reopened) != 1 {
		t.Fatalf("reopened unlearnt vocab = %d, want 1", len(reopened))
	}
	if reopened[0].Learnt {
		t.Fatalf("reopened entry Learnt = true, want false")
	}
	if reopened[0].SeenCount != 2 {
		t.Fatalf("reopened entry SeenCount = %d, want 2", reopened[0].SeenCount)
	}
}
