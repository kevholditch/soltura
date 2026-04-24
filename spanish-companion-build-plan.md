# Spanish Companion — Claude Code Build Plan

## Project Overview

A personal Spanish language learning companion web app. The user writes in Spanish, the AI responds naturally in Spanish while silently analysing errors, logging corrections to a persistent vocabulary store, and generating end-of-session summaries. Built with Go backend, SQLite persistence, HTMX frontend, and Anthropic API with streaming.

---

## Tech Stack

| Layer | Choice |
|---|---|
| Frontend | HTML + TailwindCSS (CDN) + HTMX |
| Backend | Go 1.22+ with Chi router |
| Persistence | SQLite via `modernc.org/sqlite` (pure Go, no CGo) |
| AI | Anthropic API called directly via `net/http` |
| Streaming | Server-Sent Events (SSE) |

---

## Project Structure

```
spanish-companion/
├── main.go
├── go.mod
├── go.sum
├── handlers/
│   ├── session.go
│   ├── vocabulary.go
│   └── summary.go
├── anthropic/
│   └── client.go
├── store/
│   └── sqlite.go
├── models/
│   └── models.go
├── prompts/
│   └── system.go
└── web/
    ├── index.html
    ├── app.js
    └── style.css
```

---

## Phase 1 — Project Scaffold and Database

### 1.1 Initialise the Go module

```bash
go mod init spanish-companion
go get github.com/go-chi/chi/v5
go get modernc.org/sqlite
```

### 1.2 Define models (`models/models.go`)

Create the following structs:

```go
type Session struct {
    ID        string    `json:"id"`
    Topic     string    `json:"topic"`
    StartedAt time.Time `json:"started_at"`
    EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type Turn struct {
    ID           string    `json:"id"`
    SessionID    string    `json:"session_id"`
    UserText     string    `json:"user_text"`
    AgentReply   string    `json:"agent_reply"`
    Corrections  []Correction `json:"corrections"`
    CreatedAt    time.Time `json:"created_at"`
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
```

### 1.3 SQLite store (`store/sqlite.go`)

Implement the following:

**Schema — run on startup:**

```sql
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
```

**Store methods to implement:**

```go
type Store interface {
    CreateSession(topic string) (*Session, error)
    EndSession(sessionID string) error
    GetSession(sessionID string) (*Session, error)
    
    SaveTurn(sessionID, userText, agentReply string, corrections []Correction) (*Turn, error)
    GetTurns(sessionID string) ([]Turn, error)
    
    GetCorrections(sessionID string) ([]Correction, error)
    
    UpsertVocab(corrections []Correction) error
    GetVocab(limit int) ([]VocabEntry, error)
    GetVocabCount() (int, error)
}
```

For `UpsertVocab`: on conflict (original, corrected), increment `seen_count` and update `last_seen`.

Use `github.com/google/uuid` for ID generation.

---

## Phase 2 — Anthropic Client with Streaming

### 2.1 Client (`anthropic/client.go`)

Implement a client that:

- Holds the API key (read from `ANTHROPIC_API_KEY` environment variable)
- Has a base URL of `https://api.anthropic.com/v1/messages`
- Uses model `claude-sonnet-4-20250514`
- Sends requests with header `anthropic-version: 2023-06-01`

**Message types:**

```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type Request struct {
    Model     string    `json:"model"`
    MaxTokens int       `json:"max_tokens"`
    System    string    `json:"system"`
    Messages  []Message `json:"messages"`
    Stream    bool      `json:"stream"`
}

type StreamEvent struct {
    Type  string `json:"type"`
    Delta *Delta `json:"delta,omitempty"`
}

type Delta struct {
    Type string `json:"type"`
    Text string `json:"text"`
}
```

**Two client methods:**

`StreamCompletion(ctx context.Context, system string, messages []Message, onChunk func(string)) (string, error)`
- Makes a streaming POST request
- Reads SSE events line by line
- For each `content_block_delta` event, calls `onChunk(delta.text)`
- Returns the full accumulated text when stream ends

`Complete(ctx context.Context, system string, messages []Message) (string, error)`
- Non-streaming POST request
- Returns full response text
- Used for the correction analysis (runs in goroutine alongside stream)

---

## Phase 3 — Prompts

### 3.1 System prompts (`prompts/system.go`)

**Conversation system prompt:**

```
You are a Spanish conversation partner for an advanced English speaker learning Spanish.
The user has strong comprehension (C1 level) but weaker productive/output skills.

Your role:
- Always respond ONLY in Spanish
- Keep responses conversational, natural, and engaging
- Match the user's topic and energy
- Pitch your language at high B2/C1 — rich vocabulary, varied grammar, but not academic
- Ask a follow-up question to keep the conversation flowing
- If the user writes in English, gently respond in Spanish and invite them to try in Spanish

You are currently discussing: {{.Topic}}

The conversation so far is provided in the message history.
```

**Correction analysis prompt (non-streaming, runs in parallel):**

```
You are a Spanish language correction engine. Analyse the following Spanish text written by an advanced learner and identify errors.

Text to analyse:
{{.UserText}}

Return a JSON array of corrections. Each correction object must have:
- "original": the incorrect word or phrase as written
- "corrected": the correct form
- "explanation": a brief explanation in English (1 sentence max)
- "category": one of: grammar | vocabulary | gender | spelling | register

Return ONLY the JSON array. No preamble, no markdown. If there are no errors, return an empty array [].

Example:
[
  {
    "original": "soy muy bien",
    "corrected": "estoy muy bien",
    "explanation": "Use 'estar' not 'ser' for temporary states like feeling well.",
    "category": "grammar"
  }
]
```

**Session summary prompt:**

```
You are summarising a Spanish learning session. Here is the data:

Topic: {{.Topic}}
Duration: {{.Duration}}
Number of turns: {{.TurnCount}}

Corrections made:
{{.CorrectionsJSON}}

Write a concise session summary in English with these sections:
1. What went well (1-2 sentences, genuine and specific)
2. Key corrections (group by category, max 5 most important)
3. Words to review (list the corrected forms to remember)
4. One thing to focus on next session

Tone: encouraging but honest. This person is smart and doesn't want empty praise.
```

---

## Phase 4 — HTTP Handlers

### 4.1 Session handler (`handlers/session.go`)

**POST `/api/sessions`**
- Body: `{ "topic": "string" }`
- Creates a new session in the store
- Returns: `{ "session_id": "uuid", "seed_content": "string" }`
- Generate seed content: call `anthropic.Complete` with a prompt asking for a 3-4 sentence Spanish paragraph on the topic at C1 level, ending with a question. Return this as `seed_content`.

**POST `/api/sessions/{sessionID}/turns`**
- Body: `{ "user_text": "string", "history": [{"role": "user"|"assistant", "content": "string"}] }`
- Returns: SSE stream

SSE stream implementation:
1. Set headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
2. Launch goroutine A: call `anthropic.StreamCompletion` with the conversation system prompt and full history. For each chunk, write `data: {"type":"chunk","text":"..."}` to the SSE stream and flush.
3. Launch goroutine B simultaneously: call `anthropic.Complete` with the correction analysis prompt and the user's text. Parse the returned JSON into `[]Correction`.
4. Wait for both goroutines to complete.
5. Save the turn and corrections to the store via `store.SaveTurn`.
6. Upsert vocab via `store.UpsertVocab`.
7. Send final SSE event: `data: {"type":"corrections","corrections":[...]}` 
8. Send `data: {"type":"done"}` and close.

Use a `sync.WaitGroup` or errgroup to coordinate the two goroutines. The stream chunks from goroutine A should be sent immediately as they arrive — do not wait for goroutine B.

**POST `/api/sessions/{sessionID}/end`**
- Calls `store.EndSession`
- Returns 200 OK

### 4.2 Summary handler (`handlers/summary.go`)

**GET `/api/sessions/{sessionID}/summary`**
- Fetches session, turns, and corrections from store
- Builds summary prompt with the data
- Calls `anthropic.Complete`
- Returns: `{ "summary": "string", "correction_count": int, "turn_count": int }`

### 4.3 Vocabulary handler (`handlers/vocabulary.go`)

**GET `/api/vocab`**
- Query param: `limit` (default 20)
- Returns vocab entries ordered by `seen_count DESC, last_seen DESC`
- Returns: `{ "entries": [...], "total": int }`

---

## Phase 5 — Main and Routing (`main.go`)

```go
func main() {
    // Load ANTHROPIC_API_KEY from environment
    // Initialise SQLite store (db file: ./spanish.db)
    // Run schema migrations
    // Initialise Anthropic client
    // Initialise handlers with store + client injected
    
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    
    // Serve static files
    r.Handle("/", http.FileServer(http.Dir("./web")))
    
    // API routes
    r.Post("/api/sessions", sessionHandler.Create)
    r.Post("/api/sessions/{sessionID}/turns", sessionHandler.Turn)
    r.Post("/api/sessions/{sessionID}/end", sessionHandler.End)
    r.Get("/api/sessions/{sessionID}/summary", summaryHandler.Get)
    r.Get("/api/vocab", vocabHandler.List)
    
    log.Println("Starting on :8080")
    http.ListenAndServe(":8080", r)
}
```

---

## Phase 6 — Frontend

### 6.1 Layout (`web/index.html`)

The app has three views, toggled by JS (no page reloads):

**View 1: Start screen**
- Text input for topic
- "Start Session" button
- Link to vocabulary review

**View 2: Conversation screen**
- Session topic displayed at top
- Scrollable message thread (user messages right-aligned, agent messages left-aligned)
- Seed content displayed as first agent message on load
- Text area for user input (Cmd/Ctrl+Enter to submit)
- Submit button (disabled while streaming)
- After each turn: corrections panel slides in below the agent reply showing any corrections with category badges
- "End Session" button

**View 3: Summary screen**
- Rendered summary text
- Correction count and turn count stats
- "Start New Session" button

**View 4: Vocabulary screen** (accessible any time via nav)
- Table of vocab entries: original | corrected | category | times seen
- Sorted by times seen descending

### 6.2 JavaScript (`web/app.js`)

Implement the following functions:

**`startSession(topic)`**
- POST `/api/sessions` with topic
- Store `sessionId` and `history` array in memory
- Display seed content as first message
- Add seed content to history as `{role: "assistant", content: seedContent}`
- Switch to conversation view

**`submitTurn(userText)`**
- Add user message to UI immediately
- Add to history array: `{role: "user", content: userText}`
- Clear input, disable submit button
- Open `EventSource` via fetch with POST (use fetch + ReadableStream for POST SSE, not EventSource which is GET-only)
- Read stream chunks:
  - `type: chunk` → append text to current agent message bubble (streaming effect)
  - `type: corrections` → render corrections panel beneath agent message
  - `type: done` → add completed agent reply to history, re-enable submit
- On completion, add agent reply to history: `{role: "assistant", content: fullReply}`

**`endSession()`**
- POST `/api/sessions/{sessionID}/end`
- GET `/api/sessions/{sessionID}/summary`
- Render summary view

**`loadVocab()`**
- GET `/api/vocab`
- Render vocab table

### 6.3 Styling (`web/style.css`)

Use TailwindCSS via CDN. Design aesthetic: dark theme, clean and focused. Think a refined terminal aesthetic — not garish, but with character. Spanish warm accent colour (terracotta or saffron). Clear typographic hierarchy. Corrections panel uses subtle colour coding per category (grammar = amber, vocabulary = blue, gender = purple, spelling = red, register = green).

Import Google Fonts — use **Fraunces** for the app title/headings and **JetBrains Mono** for the conversation text (gives a focused, slightly editorial feel appropriate for a writing practice tool).

---

## Phase 7 — Environment and Running

### 7.1 `.env` handling

Read `ANTHROPIC_API_KEY` from environment. Optionally support a `.env` file using a simple manual parser (avoid adding a dependency just for this):

```go
func loadEnv() {
    data, err := os.ReadFile(".env")
    if err != nil {
        return
    }
    for _, line := range strings.Split(string(data), "\n") {
        parts := strings.SplitN(line, "=", 2)
        if len(parts) == 2 {
            os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
        }
    }
}
```

### 7.2 Running

```bash
ANTHROPIC_API_KEY=your_key go run .
# or with .env file:
go run .
```

App available at `http://localhost:8080`

---

## Implementation Notes for Claude Code

- Use `github.com/google/uuid` for all ID generation
- SQLite WAL mode recommended: `PRAGMA journal_mode=WAL` on connection open
- All Anthropic API errors should be logged and returned as structured JSON errors to the frontend
- The correction goroutine (goroutine B) should not block the SSE stream under any circumstances — if it fails or times out, send an empty corrections array and log the error
- History passed in each turn request is the full conversation history from the frontend — the backend is stateless per-request, history is managed client-side
- Max history to send to Anthropic: last 20 turns (40 messages) to avoid token limits
- Set a 90-second timeout on the Anthropic streaming client context
- Correction analysis call: set `max_tokens: 1000`, should be ample for any realistic correction set

---

## Suggested Build Order

1. `models/models.go` — structs first, no dependencies
2. `store/sqlite.go` — database layer, testable in isolation
3. `anthropic/client.go` — API client with streaming
4. `prompts/system.go` — all prompt templates
5. `handlers/session.go` — core turn handler (most complex)
6. `handlers/summary.go` and `handlers/vocabulary.go`
7. `main.go` — wire everything together
8. `web/` — frontend last, once API is working

Test the API layer with `curl` before building the frontend.
