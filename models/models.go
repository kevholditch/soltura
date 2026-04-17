package models

import "time"

type Session struct {
	ID        string     `json:"id"`
	Topic     string     `json:"topic"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type Turn struct {
	ID          string       `json:"id"`
	SessionID   string       `json:"session_id"`
	UserText    string       `json:"user_text"`
	AgentReply  string       `json:"agent_reply"`
	Corrections []Correction `json:"corrections"`
	CreatedAt   time.Time    `json:"created_at"`
}

type Correction struct {
	ID          string `json:"id"`
	TurnID      string `json:"turn_id"`
	SessionID   string `json:"session_id"`
	Original    string `json:"original"`
	Corrected   string `json:"corrected"`
	Explanation string `json:"explanation"`
	Category    string `json:"category"` // grammar | vocabulary | gender | spelling | register
}

type VocabEntry struct {
	ID          string    `json:"id"`
	Original    string    `json:"original"`
	Corrected   string    `json:"corrected"`
	Explanation string    `json:"explanation"`
	Category    string    `json:"category"`
	SeenCount   int       `json:"seen_count"`
	LastSeen    time.Time `json:"last_seen"`
}
