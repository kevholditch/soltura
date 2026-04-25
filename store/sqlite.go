package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"soltura/models"
)

// Store defines the persistence interface for the Spanish companion app.
type Store interface {
	CreateSession(topic string) (*models.Session, error)
	EndSession(sessionID string) error
	GetSession(sessionID string) (*models.Session, error)
	SetSessionSeedContent(sessionID, seedContent string) error
	ListSessions(limit int) ([]models.SessionListItem, error)
	GetSessionReview(sessionID string) (*models.SessionReview, error)

	SaveTurn(sessionID, userText, agentReply string, corrections []models.Correction) (*models.Turn, error)
	GetTurns(sessionID string) ([]models.Turn, error)

	GetCorrections(sessionID string) ([]models.Correction, error)

	UpsertVocab(corrections []models.Correction) error
	GetVocab(limit int) ([]models.VocabEntry, error)
	GetVocabSorted(limit int, sort string) ([]models.VocabEntry, error)
	GetVocabByIDs(ids []string) ([]models.VocabEntry, error)
	GetVocabCount() (int, error)

	GetUnlearntVocab(limit int) ([]models.VocabEntry, error)
	MarkVocabLearnt(ids []string) error
}

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    seed_content TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS turns (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    user_text TEXT NOT NULL,
    agent_reply TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS corrections (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    original TEXT NOT NULL,
    corrected TEXT NOT NULL,
    explanation TEXT NOT NULL,
    category TEXT NOT NULL,
    FOREIGN KEY (turn_id) REFERENCES turns(id)
);

CREATE TABLE IF NOT EXISTS vocab (
    id TEXT PRIMARY KEY,
    original TEXT NOT NULL,
    corrected TEXT NOT NULL,
    explanation TEXT NOT NULL,
    category TEXT NOT NULL,
    seen_count INTEGER DEFAULT 1,
    last_seen DATETIME NOT NULL,
    UNIQUE(original, corrected)
);
`

// SQLiteStore is a SQLite-backed implementation of Store.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database at dbPath, enables WAL
// mode, runs the schema migrations, and returns a ready-to-use *SQLiteStore.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("run schema: %w", err)
	}

	var sessionSeedColCount int
	sessionSeedRow := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name='seed_content'`)
	sessionSeedRow.Scan(&sessionSeedColCount) //nolint:errcheck
	if sessionSeedColCount == 0 {
		if _, err := db.Exec(`ALTER TABLE sessions ADD COLUMN seed_content TEXT NOT NULL DEFAULT ''`); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate sessions seed_content: %w", err)
		}
	}

	// Migrate: add learnt columns to vocab if missing
	var colCount int
	row := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('vocab') WHERE name='learnt'`)
	row.Scan(&colCount) //nolint:errcheck
	if colCount == 0 {
		if _, err := db.Exec(`ALTER TABLE vocab ADD COLUMN learnt INTEGER NOT NULL DEFAULT 0`); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate vocab learnt: %w", err)
		}
		if _, err := db.Exec(`ALTER TABLE vocab ADD COLUMN learnt_at DATETIME`); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate vocab learnt_at: %w", err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateSession inserts a new session and returns it.
func (s *SQLiteStore) CreateSession(topic string) (*models.Session, error) {
	sess := &models.Session{
		ID:        uuid.New().String(),
		Topic:     topic,
		StartedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, topic, started_at) VALUES (?, ?, ?)`,
		sess.ID, sess.Topic, sess.StartedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

// EndSession sets ended_at to now for the given session.
func (s *SQLiteStore) EndSession(sessionID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`UPDATE sessions SET ended_at = ? WHERE id = ?`, now, sessionID)
	if err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return nil
}

// GetSession returns the session with the given ID.
func (s *SQLiteStore) GetSession(sessionID string) (*models.Session, error) {
	row := s.db.QueryRow(`SELECT id, topic, started_at, ended_at, seed_content FROM sessions WHERE id = ?`, sessionID)
	return scanSession(row)
}

// SetSessionSeedContent stores the assistant's opening message for later review.
func (s *SQLiteStore) SetSessionSeedContent(sessionID, seedContent string) error {
	res, err := s.db.Exec(`UPDATE sessions SET seed_content = ? WHERE id = ?`, seedContent, sessionID)
	if err != nil {
		return fmt.Errorf("set session seed content: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return nil
}

// ListSessions returns recent sessions newest first. If limit <= 0 it defaults to 50.
func (s *SQLiteStore) ListSessions(limit int) ([]models.SessionListItem, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT
			s.id,
			s.topic,
			s.started_at,
			s.ended_at,
			COUNT(DISTINCT t.id) AS turn_count,
			COUNT(c.id) AS correction_count
		FROM sessions s
		JOIN turns t ON t.session_id = s.id
		LEFT JOIN corrections c ON c.turn_id = t.id
		GROUP BY s.id, s.topic, s.started_at, s.ended_at
		ORDER BY s.started_at DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	sessions := []models.SessionListItem{}
	for rows.Next() {
		item, err := scanSessionListItemRows(rows)
		if err != nil {
			return nil, err
		}
		item.Categories, err = s.getSessionCategories(item.ID)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return sessions, nil
}

// GetSessionReview returns a session and all turns with corrections for review.
func (s *SQLiteStore) GetSessionReview(sessionID string) (*models.SessionReview, error) {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	turns, err := s.GetTurns(sessionID)
	if err != nil {
		return nil, err
	}

	review := &models.SessionReview{
		Session:    *session,
		Turns:      turns,
		Categories: []string{},
	}
	if review.Turns == nil {
		review.Turns = []models.Turn{}
	}
	seenCategories := make(map[string]bool)
	for _, turn := range turns {
		for _, correction := range turn.Corrections {
			review.CorrectionCount++
			if correction.Category != "" && !seenCategories[correction.Category] {
				seenCategories[correction.Category] = true
				review.Categories = append(review.Categories, correction.Category)
			}
		}
	}
	return review, nil
}

func scanSession(row *sql.Row) (*models.Session, error) {
	var sess models.Session
	var startedAtStr string
	var endedAtStr sql.NullString
	var seedContent string

	if err := row.Scan(&sess.ID, &sess.Topic, &startedAtStr, &endedAtStr, &seedContent); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}

	t, err := time.Parse(time.RFC3339, startedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse started_at: %w", err)
	}
	sess.StartedAt = t

	if endedAtStr.Valid {
		et, err := time.Parse(time.RFC3339, endedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse ended_at: %w", err)
		}
		sess.EndedAt = &et
	}
	sess.SeedContent = seedContent

	return &sess, nil
}

func scanSessionListItemRows(rows *sql.Rows) (models.SessionListItem, error) {
	var item models.SessionListItem
	var startedAtStr string
	var endedAtStr sql.NullString

	if err := rows.Scan(&item.ID, &item.Topic, &startedAtStr, &endedAtStr, &item.TurnCount, &item.CorrectionCount); err != nil {
		return item, fmt.Errorf("scan session list item: %w", err)
	}

	t, err := time.Parse(time.RFC3339, startedAtStr)
	if err != nil {
		return item, fmt.Errorf("parse started_at: %w", err)
	}
	item.StartedAt = t

	if endedAtStr.Valid {
		et, err := time.Parse(time.RFC3339, endedAtStr.String)
		if err != nil {
			return item, fmt.Errorf("parse ended_at: %w", err)
		}
		item.EndedAt = &et
	}

	return item, nil
}

func (s *SQLiteStore) getSessionCategories(sessionID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT category FROM corrections WHERE session_id = ? ORDER BY rowid ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query session categories: %w", err)
	}
	defer rows.Close()

	categories := []string{}
	seen := make(map[string]bool)
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan session category: %w", err)
		}
		if category != "" && !seen[category] {
			seen[category] = true
			categories = append(categories, category)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session categories: %w", err)
	}
	return categories, nil
}

// SaveTurn inserts a turn and its corrections, then returns the populated Turn.
func (s *SQLiteStore) SaveTurn(sessionID, userText, agentReply string, corrections []models.Correction) (*models.Turn, error) {
	turn := &models.Turn{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		UserText:   userText,
		AgentReply: agentReply,
		CreatedAt:  time.Now().UTC(),
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.Exec(
		`INSERT INTO turns (id, session_id, user_text, agent_reply, created_at) VALUES (?, ?, ?, ?, ?)`,
		turn.ID, turn.SessionID, turn.UserText, turn.AgentReply, turn.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert turn: %w", err)
	}

	savedCorrections := make([]models.Correction, 0, len(corrections))
	for _, c := range corrections {
		c.ID = uuid.New().String()
		c.TurnID = turn.ID
		c.SessionID = sessionID
		_, err = tx.Exec(
			`INSERT INTO corrections (id, turn_id, session_id, original, corrected, explanation, category) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.ID, c.TurnID, c.SessionID, c.Original, c.Corrected, c.Explanation, c.Category,
		)
		if err != nil {
			return nil, fmt.Errorf("insert correction: %w", err)
		}
		savedCorrections = append(savedCorrections, c)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit turn: %w", err)
	}

	turn.Corrections = savedCorrections
	return turn, nil
}

// GetTurns returns all turns for a session, each with its corrections populated.
func (s *SQLiteStore) GetTurns(sessionID string) ([]models.Turn, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, user_text, agent_reply, created_at FROM turns WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query turns: %w", err)
	}
	defer rows.Close()

	var turns []models.Turn
	for rows.Next() {
		var t models.Turn
		var createdAtStr string
		if err := rows.Scan(&t.ID, &t.SessionID, &t.UserText, &t.AgentReply, &createdAtStr); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		ct, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse turn created_at: %w", err)
		}
		t.CreatedAt = ct
		turns = append(turns, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate turns: %w", err)
	}

	// Fetch corrections for each turn.
	for i := range turns {
		corrections, err := s.getTurnCorrections(turns[i].ID)
		if err != nil {
			return nil, err
		}
		turns[i].Corrections = corrections
	}

	return turns, nil
}

func (s *SQLiteStore) getTurnCorrections(turnID string) ([]models.Correction, error) {
	rows, err := s.db.Query(
		`SELECT id, turn_id, session_id, original, corrected, explanation, category FROM corrections WHERE turn_id = ? ORDER BY rowid ASC`,
		turnID,
	)
	if err != nil {
		return nil, fmt.Errorf("query corrections for turn %s: %w", turnID, err)
	}
	defer rows.Close()

	var corrections []models.Correction
	for rows.Next() {
		var c models.Correction
		if err := rows.Scan(&c.ID, &c.TurnID, &c.SessionID, &c.Original, &c.Corrected, &c.Explanation, &c.Category); err != nil {
			return nil, fmt.Errorf("scan correction: %w", err)
		}
		corrections = append(corrections, c)
	}
	return corrections, rows.Err()
}

// GetCorrections returns all corrections for a session.
func (s *SQLiteStore) GetCorrections(sessionID string) ([]models.Correction, error) {
	rows, err := s.db.Query(
		`SELECT id, turn_id, session_id, original, corrected, explanation, category FROM corrections WHERE session_id = ? ORDER BY rowid ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query corrections: %w", err)
	}
	defer rows.Close()

	var corrections []models.Correction
	for rows.Next() {
		var c models.Correction
		if err := rows.Scan(&c.ID, &c.TurnID, &c.SessionID, &c.Original, &c.Corrected, &c.Explanation, &c.Category); err != nil {
			return nil, fmt.Errorf("scan correction: %w", err)
		}
		corrections = append(corrections, c)
	}
	return corrections, rows.Err()
}

// UpsertVocab inserts or updates vocab entries derived from corrections.
func (s *SQLiteStore) UpsertVocab(corrections []models.Correction) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range corrections {
		_, err := s.db.Exec(
			`INSERT INTO vocab (id, original, corrected, explanation, category, seen_count, last_seen)
             VALUES (?, ?, ?, ?, ?, 1, ?)
             ON CONFLICT(original, corrected) DO UPDATE SET
                 seen_count = seen_count + 1,
                 last_seen = excluded.last_seen,
                 learnt = 0,
                 learnt_at = NULL`,
			uuid.New().String(), c.Original, c.Corrected, c.Explanation, c.Category, now,
		)
		if err != nil {
			return fmt.Errorf("upsert vocab: %w", err)
		}
	}
	return nil
}

// GetVocab returns vocab entries ordered by seen_count DESC, last_seen DESC.
// If limit <= 0 it defaults to 20.
func (s *SQLiteStore) GetVocab(limit int) ([]models.VocabEntry, error) {
	return s.GetVocabSorted(limit, "frequency")
}

// GetVocabSorted returns vocab entries with an explicit sort order.
func (s *SQLiteStore) GetVocabSorted(limit int, sort string) ([]models.VocabEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	orderBy := `last_seen DESC, seen_count DESC`
	if sort == "frequency" {
		orderBy = `seen_count DESC, last_seen DESC`
	}
	rows, err := s.db.Query(
		`SELECT id, original, corrected, explanation, category, seen_count, last_seen, learnt, learnt_at FROM vocab ORDER BY `+orderBy+` LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query vocab: %w", err)
	}
	defer rows.Close()

	var entries []models.VocabEntry
	for rows.Next() {
		entry, err := scanVocabEntryRows(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetVocabByIDs returns unlearnt vocab entries matching the requested IDs.
func (s *SQLiteStore) GetVocabByIDs(ids []string) ([]models.VocabEntry, error) {
	if len(ids) == 0 {
		return []models.VocabEntry{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	rows, err := s.db.Query(
		`SELECT id, original, corrected, explanation, category, seen_count, last_seen, learnt, learnt_at
		FROM vocab
		WHERE learnt = 0 AND id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY last_seen DESC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query vocab by ids: %w", err)
	}
	defer rows.Close()

	var entries []models.VocabEntry
	for rows.Next() {
		entry, err := scanVocabEntryRows(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vocab by ids: %w", err)
	}
	return entries, nil
}

func scanVocabEntryRows(rows *sql.Rows) (models.VocabEntry, error) {
	var entry models.VocabEntry
	var lastSeenStr string
	var learntInt int
	var learntAtStr sql.NullString

	if err := rows.Scan(&entry.ID, &entry.Original, &entry.Corrected, &entry.Explanation, &entry.Category, &entry.SeenCount, &lastSeenStr, &learntInt, &learntAtStr); err != nil {
		return entry, fmt.Errorf("scan vocab: %w", err)
	}

	lastSeen, err := time.Parse(time.RFC3339, lastSeenStr)
	if err != nil {
		return entry, fmt.Errorf("parse last_seen: %w", err)
	}
	entry.LastSeen = lastSeen
	entry.Learnt = learntInt != 0

	if learntAtStr.Valid {
		learntAt, err := time.Parse(time.RFC3339, learntAtStr.String)
		if err != nil {
			return entry, fmt.Errorf("parse learnt_at: %w", err)
		}
		entry.LearntAt = &learntAt
	}

	return entry, nil
}

// GetVocabCount returns the total number of distinct vocab entries.
func (s *SQLiteStore) GetVocabCount() (int, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM vocab`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count vocab: %w", err)
	}
	return count, nil
}

// GetUnlearntVocab returns unlearnt vocab ordered by seen_count DESC.
func (s *SQLiteStore) GetUnlearntVocab(limit int) ([]models.VocabEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id, original, corrected, explanation, category, seen_count, last_seen FROM vocab WHERE learnt = 0 ORDER BY seen_count DESC, last_seen DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query unlearnt vocab: %w", err)
	}
	defer rows.Close()

	var entries []models.VocabEntry
	for rows.Next() {
		var e models.VocabEntry
		var lastSeenStr string
		if err := rows.Scan(&e.ID, &e.Original, &e.Corrected, &e.Explanation, &e.Category, &e.SeenCount, &lastSeenStr); err != nil {
			return nil, fmt.Errorf("scan unlearnt vocab: %w", err)
		}
		ls, err := time.Parse(time.RFC3339, lastSeenStr)
		if err != nil {
			return nil, fmt.Errorf("parse last_seen: %w", err)
		}
		e.LastSeen = ls
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// MarkVocabLearnt sets learnt=1 and learnt_at=now for the given IDs.
func (s *SQLiteStore) MarkVocabLearnt(ids []string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, id := range ids {
		if _, err := s.db.Exec(`UPDATE vocab SET learnt = 1, learnt_at = ? WHERE id = ?`, now, id); err != nil {
			return fmt.Errorf("mark vocab learnt %s: %w", id, err)
		}
	}
	return nil
}
