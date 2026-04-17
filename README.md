# Soltura

A personal Spanish language learning companion. You write in Spanish, the AI responds naturally in Spanish, silently analyses your errors, and logs corrections to a persistent vocabulary store. At the end of each session you get a concise summary of what went well and what to focus on next.

## Features

- Streaming conversation in Spanish at C1/B2 level
- Silent error analysis running in parallel with the conversation stream
- Persistent vocabulary store — repeated mistakes are tracked by frequency
- End-of-session summary with corrections grouped by category
- Vocabulary review screen showing all corrections sorted by times seen

## Tech Stack

| Layer | Choice |
|---|---|
| Frontend | HTML + TailwindCSS (CDN) + vanilla JS |
| Backend | Go with Chi router |
| Persistence | SQLite (pure Go, no CGo) |
| AI | Anthropic API (claude-sonnet-4-6) |
| Streaming | Server-Sent Events |

## Prerequisites

- Go 1.22+
- An [Anthropic API key](https://console.anthropic.com/)

## Setup

```bash
git clone https://github.com/kevholditch/soltura.git
cd soltura

# Create a .env file with your API key
echo "ANTHROPIC_API_KEY=your_key_here" > .env

# Run
go run .
```

Open [http://localhost:8080](http://localhost:8080).

## Usage

1. **Start a session** — enter a topic (e.g. "my weekend plans", "favourite films") and click Start
2. **Converse** — write in Spanish, press Cmd+Enter (or the Send button) to submit
3. **Review corrections** — any errors appear below each agent reply with category badges
4. **End the session** — click End Session to get a summary and stats
5. **Vocabulary** — click Vocabulary in the nav to see all corrections sorted by frequency

## Project Structure

```
.
├── main.go               # Server entrypoint, routing
├── anthropic/client.go   # Anthropic API client (streaming + non-streaming)
├── handlers/
│   ├── session.go        # Session create, SSE turn stream, end
│   ├── summary.go        # Session summary generation
│   └── vocabulary.go     # Vocabulary list endpoint
├── models/models.go      # Shared structs
├── prompts/system.go     # Prompt templates
├── store/sqlite.go       # SQLite persistence layer
└── web/
    ├── index.html        # Single-page app shell
    ├── app.js            # Frontend logic
    └── style.css         # Component styles
```

## Correction Categories

| Category | Colour | Description |
|---|---|---|
| grammar | amber | Wrong tense, mood, conjugation |
| vocabulary | blue | Wrong word choice |
| gender | purple | Noun/adjective gender agreement |
| spelling | red | Spelling errors |
| register | green | Formality level mismatch |
