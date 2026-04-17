package store

import (
	"database/sql"
	"fmt"
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

	SaveTurn(sessionID, userText, agentReply string, corrections []models.Correction) (*models.Turn, error)
	GetTurns(sessionID string) ([]models.Turn, error)

	GetCorrections(sessionID string) ([]models.Correction, error)

	UpsertVocab(corrections []models.Correction) error
	GetVocab(limit int) ([]models.VocabEntry, error)
	GetVocabCount() (int, error)
}

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME
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
	row := s.db.QueryRow(`SELECT id, topic, started_at, ended_at FROM sessions WHERE id = ?`, sessionID)
	return scanSession(row)
}

func scanSession(row *sql.Row) (*models.Session, error) {
	var sess models.Session
	var startedAtStr string
	var endedAtStr sql.NullString

	if err := row.Scan(&sess.ID, &sess.Topic, &startedAtStr, &endedAtStr); err != nil {
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

	return &sess, nil
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
		`SELECT id, turn_id, session_id, original, corrected, explanation, category FROM corrections WHERE turn_id = ?`,
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
		`SELECT id, turn_id, session_id, original, corrected, explanation, category FROM corrections WHERE session_id = ?`,
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
                 last_seen = excluded.last_seen`,
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
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, original, corrected, explanation, category, seen_count, last_seen FROM vocab ORDER BY seen_count DESC, last_seen DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query vocab: %w", err)
	}
	defer rows.Close()

	var entries []models.VocabEntry
	for rows.Next() {
		var e models.VocabEntry
		var lastSeenStr string
		if err := rows.Scan(&e.ID, &e.Original, &e.Corrected, &e.Explanation, &e.Category, &e.SeenCount, &lastSeenStr); err != nil {
			return nil, fmt.Errorf("scan vocab: %w", err)
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

// GetVocabCount returns the total number of distinct vocab entries.
func (s *SQLiteStore) GetVocabCount() (int, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM vocab`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count vocab: %w", err)
	}
	return count, nil
}
