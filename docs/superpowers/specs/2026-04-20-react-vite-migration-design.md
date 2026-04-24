# React + Vite Frontend Migration

**Date:** 2026-04-20  
**Status:** Approved

## Context

The current frontend is vanilla HTML/CSS/JS in `web/`. Upcoming features (tabbed panels, drill mode, trend analysis, E2E testing) require a proper component-based framework. This spec covers a pure tech swap — identical UI and functionality, no new features.

## Goals

- Replace vanilla JS with React + Vite
- Preserve all existing functionality exactly
- Add Make tasks covering dev (hot-reload) and production workflows
- Default to local Gemma model; Anthropic is an explicit override

## Directory Structure

```
frontend/                  ← React source (new)
  src/
    main.jsx               ← ReactDOM.createRoot entry point
    App.jsx                ← view state machine (start/conversation/summary/vocab)
    index.css              ← existing animations + Tailwind directives
    views/
      StartView.jsx        ← topic input + start button
      ConversationView.jsx ← message thread, input, SSE streaming
      SummaryView.jsx      ← session stats + summary text
      VocabView.jsx        ← vocabulary table
    components/
      MessageBubble.jsx    ← user and agent bubbles, loading state, thinking timer
      CorrectionsPanel.jsx ← grammar corrections below agent messages
      LoadingBubble.jsx    ← animated dots + elapsed timer
  index.html
  package.json
  vite.config.js           ← proxy /api → :8080, build output to ../web
web/                       ← Vite build output (Go serves this — unchanged)
docs/
Makefile                   ← updated (see below)
```

## Component Responsibilities

| Component | Owns |
|---|---|
| `App.jsx` | Current view, sessionId, history array, view transitions |
| `StartView.jsx` | Topic input, calls `startSession()`, error display |
| `ConversationView.jsx` | Message list, `submitTurn()` SSE logic, corrections |
| `SummaryView.jsx` | Fetches + displays summary after session end |
| `VocabView.jsx` | Fetches + displays vocab table |
| `MessageBubble.jsx` | Renders user/agent bubble, streaming class, markdown |
| `CorrectionsPanel.jsx` | Renders corrections list below a message |
| `LoadingBubble.jsx` | Animated dots + thinking timer (used in start + turns) |

State is managed with `useState` + props. No external state library.

## Data Flow

```
App.jsx (sessionId, history, currentView)
  └── ConversationView (receives sessionId, history, onHistoryUpdate, onEnd)
        └── SSE fetch → chunks → MessageBubble updates in place
        └── corrections event → CorrectionsPanel appended below bubble
```

## API Integration

No backend changes. React calls the same endpoints:

- `POST /api/sessions` — session creation (blocking, shows LoadingBubble with timer)
- `POST /api/sessions/:id/turns` — SSE stream (chunks → markdown render)
- `POST /api/sessions/:id/end`
- `GET /api/sessions/:id/summary`
- `GET /api/vocab?limit=50`

## Styling

- Tailwind CSS via Vite/PostCSS (replaces CDN `<script>` tag)
- Existing custom animations (`blink`, `pulse-border`, `.thinking-timer`) migrated to `index.css`
- No visual changes — pixel-for-pixel match with current UI

## Vite Configuration

```js
// frontend/vite.config.js — root defaults to this file's directory
export default {
  build: { outDir: '../web', emptyOutDir: true },
  server: {
    proxy: { '/api': 'http://localhost:8080' }
  }
}
```

## Makefile

```makefile
OLLAMA_MODEL := gemma3:12b
BACKEND      := $(filter anthropic,$(MAKECMDGOALS))

.PHONY: install-model run anthropic build dev build-frontend

anthropic:
	@:

install-model:
	# unchanged from current

## run [anthropic]: build frontend then start Go (default: Gemma)
run: build-frontend
	@if [ "$(BACKEND)" = "anthropic" ]; then \
		go run .; \
	else \
		LLM_BACKEND=ollama go run .; \
	fi

## dev [anthropic]: Vite hot-reload + Go (default: Gemma)
dev: frontend/node_modules
	@if [ "$(BACKEND)" = "anthropic" ]; then \
		(trap 'kill %1 2>/dev/null' EXIT; cd frontend && npm run dev & go run .); \
	else \
		(trap 'kill %1 2>/dev/null' EXIT; cd frontend && npm run dev & LLM_BACKEND=ollama go run .); \
	fi

## build-frontend: compile React into web/
build-frontend: frontend/node_modules
	cd frontend && npm run build

## build: compile React + Go binary
build: build-frontend
	go build -o soltura .

frontend/node_modules:
	cd frontend && npm install
```

## Migration Approach

1. Scaffold `frontend/` with Vite + React + Tailwind
2. Translate each view/component from `app.js` one at a time, verifying each against the current UI
3. Migrate SSE streaming logic into `ConversationView`
4. Update Makefile
5. Delete `web/index.html`, `web/app.js`, `web/style.css` (replaced by Vite output)
6. Add `web/` to `.gitignore` (it's now generated)

## Verification

- `make dev` → opens on `:5173`, all four views work, SSE streams correctly, loading timer appears
- `make run` → opens on `:8080` via Go, identical behaviour
- `make run anthropic` → same but uses Anthropic API
- `make build` → compiles without errors, binary serves the built frontend
