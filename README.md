# Soltura

Soltura is a Spanish language learning companion built around realistic conversation plus targeted practice. You chat in Spanish, get silent correction analysis, track recurring mistakes in SQLite, and drill weak patterns until they are mastered.

## Current Features

- Real-time Spanish conversation over SSE with incremental assistant streaming
- Parallel correction analysis per turn (grammar/vocabulary/gender/spelling/register)
- Corrections timeline in-chat plus vocabulary review across sessions
- Session summary generation at end of conversation
- Drill mode:
  - pattern selection from unlearnt vocab
  - quick correctness mark
  - streamed coaching feedback
  - mastery tracking and transition to next pattern
- Deterministic `LLM_BACKEND=test` mode with fixture-driven responses for full e2e testing

## Tech Stack

| Layer | Choice |
|---|---|
| Frontend | React 18 + Vite + TailwindCSS |
| Backend | Go + Chi router |
| Persistence | SQLite (`modernc.org/sqlite`, no CGO) |
| LLM Backends | Anthropic, Ollama, deterministic fixture-backed test backend |
| Streaming | Server-Sent Events |
| E2E Testing | Playwright |
| CI | GitHub Actions (`.github/workflows/tests.yml`) |

## LLM Backends

`LLM_BACKEND` supports:

- `anthropic` (default when running `go run .`)
- `ollama`
- `test` (fixture-backed deterministic mode for testing)

Anthropic supports strong/fast model lanes:

- strong: `ANTHROPIC_MODEL_STRONG` (default `claude-sonnet-4-6`)
- fast: `ANTHROPIC_MODEL_FAST` (default `claude-haiku-4-5`)

## Configuration

Common environment variables:

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `./spanish.db` | SQLite file path |
| `LLM_BACKEND` | `anthropic` | `anthropic`, `ollama`, or `test` |
| `ANTHROPIC_API_KEY` | (none) | Required for `LLM_BACKEND=anthropic` |
| `ANTHROPIC_MODEL_STRONG` | `claude-sonnet-4-6` | Optional |
| `ANTHROPIC_MODEL_FAST` | `claude-haiku-4-5` | Optional |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Used for `LLM_BACKEND=ollama` |
| `OLLAMA_MODEL` | `gemma3:12b` | Used for `LLM_BACKEND=ollama` |
| `TEST_FIXTURE_PATH` | `testdata/llm/default.json` | Used for `LLM_BACKEND=test` |

## Local Development

Prerequisites:

- Go 1.22+
- Node.js 20+ (for frontend build + Playwright e2e)

Clone:

```bash
git clone https://github.com/kevholditch/soltura.git
cd soltura
```

Run options:

```bash
# Option 1: Anthropic backend (go run defaults to anthropic)
ANTHROPIC_API_KEY=your_key_here go run .

# Option 2: Local Ollama backend (via Make)
make run

# Option 3: Vite dev + backend
make dev
```

Open [http://localhost:8080](http://localhost:8080).

## Testing

```bash
# Go unit tests
make test-go

# Install Playwright deps + Chromium
make install-playwright

# Playwright e2e
make test-e2e

# All tests
make test-all
```

The CI workflow runs `make install-playwright` and `make test-all` on every push and pull request.

## Main Make Targets

- `make build` — build frontend and Go binary
- `make run` — run app with Ollama backend
- `make run anthropic` — run app with Anthropic backend
- `make dev` — Vite + backend local dev
- `make install-model` — install/start Ollama and pull configured model
- `make test-go` — Go test suite
- `make install-playwright` — Playwright/browser setup
- `make test-e2e` — Playwright e2e suite
- `make test-all` — combined tests

## API Endpoints

- `POST /api/sessions`
- `POST /api/sessions/{sessionID}/turns` (SSE)
- `POST /api/sessions/{sessionID}/end`
- `GET /api/sessions/{sessionID}/summary`
- `GET /api/vocab`
- `GET /api/drills/phrases`
- `POST /api/drills/start`
- `POST /api/drills/turn` (SSE)

## Project Layout

```text
.
├── main.go
├── handlers/
├── store/
├── llm/
├── anthropic/
├── ollama/
├── testllm/             # deterministic fixture-backed backend
├── testdata/llm/        # fixture scripts for test backend
├── frontend/            # React + Vite app + Playwright tests
├── web/                 # built frontend output
└── .github/workflows/   # CI workflows
```
