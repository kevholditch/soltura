# React + Vite Frontend Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the vanilla HTML/CSS/JS frontend with React + Vite while preserving identical UI and functionality.

**Architecture:** React source lives in `frontend/src/`, Vite builds to `web/` (Go server unchanged). During dev, Vite runs on `:5173` and proxies `/api/*` to Go on `:8080`. In production `make run` builds first then starts Go on `:8080`.

**Tech Stack:** React 18, Vite 5, Tailwind CSS 3 (via PostCSS), marked (npm), `@vitejs/plugin-react`

---

## File Map

### Created
- `frontend/package.json` — npm deps and scripts
- `frontend/vite.config.js` — build output `../web`, dev proxy `/api → :8080`
- `frontend/tailwind.config.js` — content paths + custom colours (terracotta, saffron)
- `frontend/postcss.config.js` — tailwind + autoprefixer
- `frontend/index.html` — entry HTML with Google Fonts, no CDN scripts
- `frontend/src/main.jsx` — ReactDOM.createRoot entry point
- `frontend/src/index.css` — Tailwind directives + all custom CSS from `web/style.css`
- `frontend/src/App.jsx` — view state machine (start / conversation / summary / vocab)
- `frontend/src/views/StartView.jsx` — topic input form, session creation, loading bubble
- `frontend/src/views/ConversationView.jsx` — message list, SSE streaming, input
- `frontend/src/views/SummaryView.jsx` — fetches + displays session summary
- `frontend/src/views/VocabView.jsx` — fetches + displays vocab table
- `frontend/src/components/Message.jsx` — renders user or agent message + corrections
- `frontend/src/components/CorrectionsPanel.jsx` — corrections list below agent message
- `frontend/src/components/LoadingBubble.jsx` — animated dots + elapsed timer

### Modified
- `Makefile` — add `dev`, `build-frontend`, `frontend/node_modules` targets; default to Gemma
- `.gitignore` — add `web/` (now generated) and `frontend/node_modules`

### Deleted
- `web/index.html`, `web/app.js`, `web/style.css` — replaced by Vite build output

---

## Task 1: Scaffold frontend/ with Vite, React, and Tailwind

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.js`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/postcss.config.js`
- Create: `frontend/index.html`
- Create: `frontend/src/main.jsx`
- Create: `frontend/src/index.css`

- [ ] **Step 1: Create frontend/package.json**

```json
{
  "name": "soltura-frontend",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "marked": "^12.0.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.3.1",
    "autoprefixer": "^10.4.20",
    "postcss": "^8.4.47",
    "tailwindcss": "^3.4.14",
    "vite": "^5.4.10"
  }
}
```

- [ ] **Step 2: Create frontend/vite.config.js**

```js
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: { outDir: '../web', emptyOutDir: true },
  server: {
    proxy: { '/api': 'http://localhost:8080' }
  }
})
```

- [ ] **Step 3: Create frontend/tailwind.config.js**

```js
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        terracotta: '#C1440E',
        saffron: '#F5A623',
      },
    },
  },
  plugins: [],
}
```

- [ ] **Step 4: Create frontend/postcss.config.js**

```js
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 5: Create frontend/index.html**

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Soltura — Spanish Companion</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Fraunces:ital,opsz,wght@0,9..144,400;0,9..144,600;0,9..144,700;1,9..144,400&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
</head>
<body>
  <div id="root"></div>
  <script type="module" src="/src/main.jsx"></script>
</body>
</html>
```

- [ ] **Step 6: Create frontend/src/main.jsx**

```jsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

- [ ] **Step 7: Create frontend/src/index.css — Tailwind directives + all custom CSS**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* ── Fonts ─────────────────────────────────────────────── */

.font-fraunces,
h1, h2, h3 {
  font-family: 'Fraunces', serif;
}

.font-mono,
textarea,
input[type="text"],
.agent-bubble,
.user-bubble,
table,
.correction-item,
.corrections-heading,
.vocab-row td {
  font-family: 'JetBrains Mono', monospace;
}

/* ── Scrollbar ─────────────────────────────────────────── */

.message-thread::-webkit-scrollbar { width: 4px; }
.message-thread::-webkit-scrollbar-track { background: transparent; }
.message-thread::-webkit-scrollbar-thumb { background: #374151; border-radius: 2px; }
.message-thread::-webkit-scrollbar-thumb:hover { background: #4b5563; }

/* ── Messages ──────────────────────────────────────────── */

.user-message {
  display: flex;
  justify-content: flex-end;
}

.agent-message {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  max-width: 85%;
}

.user-bubble {
  background-color: #2a1a0e;
  border: 1px solid #7c2d12;
  color: #fed7aa;
  padding: 10px 14px;
  border-radius: 16px 16px 4px 16px;
  font-size: 0.875rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-width: 85%;
}

.agent-bubble {
  background-color: #111827;
  border: 1px solid #1f2937;
  color: #e5e7eb;
  padding: 12px 16px;
  border-radius: 4px 16px 16px 16px;
  font-size: 0.875rem;
  line-height: 1.7;
  word-break: break-word;
  width: 100%;
}

.agent-bubble p { margin: 0 0 0.75em; }
.agent-bubble p:last-child { margin-bottom: 0; }
.agent-bubble strong { color: #f9fafb; font-weight: 600; }
.agent-bubble em { color: #d1d5db; }
.agent-bubble ul, .agent-bubble ol { margin: 0 0 0.75em 1.25em; }
.agent-bubble li { margin-bottom: 0.25em; }
.agent-bubble code { background: #1f2937; padding: 1px 5px; border-radius: 4px; font-size: 0.8em; }

.summary-text p { margin: 0 0 0.75em; }
.summary-text p:last-child { margin-bottom: 0; }
.summary-text strong { color: #f9fafb; font-weight: 600; }
.summary-text ul, .summary-text ol { margin: 0 0 0.75em 1.25em; }
.summary-text li { margin-bottom: 0.25em; }

.error-bubble {
  background-color: #1c0a0a;
  border-color: #7f1d1d;
  color: #fca5a5;
}

.error-message {
  text-align: center;
  color: #f87171;
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.75rem;
  padding: 8px 0;
  opacity: 0.8;
}

/* ── Loading dots ──────────────────────────────────────── */

.loading-bubble { min-width: 60px; }

.thinking-timer {
  margin-left: 8px;
  font-size: 0.75rem;
  color: #6b7280;
}

.loading-dots span { animation: blink 1.4s infinite both; }
.loading-dots span:nth-child(2) { animation-delay: 0.2s; }
.loading-dots span:nth-child(3) { animation-delay: 0.4s; }

@keyframes blink {
  0%, 80%, 100% { opacity: 0; }
  40% { opacity: 1; }
}

/* ── Streaming pulse ───────────────────────────────────── */

@keyframes pulse-border {
  0%, 100% { border-color: #1f2937; box-shadow: none; }
  50% { border-color: #c1440e55; box-shadow: 0 0 0 2px #c1440e18; }
}

.streaming { animation: pulse-border 1.4s ease-in-out infinite; }

/* ── Corrections panel ─────────────────────────────────── */

.corrections-panel {
  width: 100%;
  background-color: #0d1117;
  border: 1px solid #1f2937;
  border-top: 2px solid #c1440e44;
  border-radius: 0 0 12px 12px;
  padding: 12px 16px;
  animation: slide-in 0.2s ease-out;
}

@keyframes slide-in {
  from { opacity: 0; transform: translateY(-6px); }
  to { opacity: 1; transform: translateY(0); }
}

.corrections-heading {
  font-size: 0.65rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #6b7280;
  margin-bottom: 10px;
  font-weight: 500;
}

.correction-item {
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid #1a2030;
}

.correction-item:last-child { margin-bottom: 0; padding-bottom: 0; border-bottom: none; }

.correction-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 5px;
}

.correction-original { font-size: 0.8rem; color: #6b7280; text-decoration: line-through; text-decoration-color: #4b5563; }
.correction-arrow { font-size: 0.75rem; color: #374151; }
.correction-corrected { font-size: 0.8rem; color: #a7f3d0; font-weight: 500; }
.correction-explanation { font-size: 0.75rem; color: #9ca3af; line-height: 1.5; margin: 0; }

/* ── Category badges ───────────────────────────────────── */

.correction-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 500;
  font-family: 'JetBrains Mono', monospace;
  text-transform: lowercase;
}

.category-grammar    { background: #92400e; color: #fde68a; }
.category-vocabulary { background: #1e3a5f; color: #93c5fd; }
.category-gender     { background: #4a1d96; color: #ddd6fe; }
.category-spelling   { background: #7f1d1d; color: #fca5a5; }
.category-register   { background: #14532d; color: #86efac; }

/* ── Vocabulary table ──────────────────────────────────── */

.vocab-row:hover td { background-color: #0f172a; }
.vocab-row td { font-size: 0.8rem; transition: background-color 0.1s ease; }

/* ── Nav ───────────────────────────────────────────────── */

nav { backdrop-filter: blur(8px); }

/* ── Responsive tweaks ─────────────────────────────────── */

@media (max-width: 640px) {
  .agent-message, .user-message { max-width: 100%; }
  .user-bubble { max-width: 90%; }
}
```

- [ ] **Step 8: Install dependencies and verify build works**

```bash
cd frontend && npm install
npm run build
```

Expected: `web/index.html` and `web/assets/` are created with no errors.

- [ ] **Step 9: Commit scaffold**

```bash
git add frontend/ web/
git commit -m "chore: scaffold React+Vite frontend with Tailwind"
```

---

## Task 2: LoadingBubble component

**Files:**
- Create: `frontend/src/components/LoadingBubble.jsx`

- [ ] **Step 1: Create frontend/src/components/LoadingBubble.jsx**

```jsx
import { useState, useEffect } from 'react'

export default function LoadingBubble() {
  const [seconds, setSeconds] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => setSeconds(s => s + 1), 1000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="agent-bubble loading-bubble">
      <span className="loading-dots">
        <span>.</span><span>.</span><span>.</span>
      </span>
      <span className="thinking-timer">{seconds}s</span>
    </div>
  )
}
```

- [ ] **Step 2: Verify build still passes**

```bash
cd frontend && npm run build
```

Expected: no errors.

---

## Task 3: CorrectionsPanel component

**Files:**
- Create: `frontend/src/components/CorrectionsPanel.jsx`

- [ ] **Step 1: Create frontend/src/components/CorrectionsPanel.jsx**

```jsx
export default function CorrectionsPanel({ corrections }) {
  if (!corrections || corrections.length === 0) return null

  return (
    <div className="corrections-panel">
      <div className="corrections-heading">
        {corrections.length === 1 ? '1 correction' : `${corrections.length} corrections`}
      </div>
      {corrections.map((c, i) => (
        <div key={i} className="correction-item">
          <div className="correction-row">
            <span className="correction-original">{c.original}</span>
            <span className="correction-arrow">→</span>
            <span className="correction-corrected">{c.corrected}</span>
            <span className={`correction-badge category-${c.category}`}>{c.category}</span>
          </div>
          <p className="correction-explanation">{c.explanation}</p>
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 2: Verify build still passes**

```bash
cd frontend && npm run build
```

---

## Task 4: Message component

**Files:**
- Create: `frontend/src/components/Message.jsx`

- [ ] **Step 1: Create frontend/src/components/Message.jsx**

```jsx
import { marked } from 'marked'
import LoadingBubble from './LoadingBubble.jsx'
import CorrectionsPanel from './CorrectionsPanel.jsx'

export default function Message({ role, text, corrections = [], isLoading = false, isStreaming = false, isError = false }) {
  if (role === 'user') {
    return (
      <div className="user-message">
        <div className="user-bubble">{text}</div>
      </div>
    )
  }

  return (
    <div className="agent-message">
      {isLoading ? (
        <LoadingBubble />
      ) : (
        <div
          className={`agent-bubble${isStreaming ? ' streaming' : ''}${isError ? ' error-bubble' : ''}`}
          dangerouslySetInnerHTML={{ __html: marked.parse(text || '') }}
        />
      )}
      <CorrectionsPanel corrections={corrections} />
    </div>
  )
}
```

- [ ] **Step 2: Verify build still passes**

```bash
cd frontend && npm run build
```

---

## Task 5: App.jsx view state machine

**Files:**
- Create: `frontend/src/App.jsx`

- [ ] **Step 1: Create frontend/src/App.jsx**

```jsx
import { useState } from 'react'
import StartView from './views/StartView.jsx'
import ConversationView from './views/ConversationView.jsx'
import SummaryView from './views/SummaryView.jsx'
import VocabView from './views/VocabView.jsx'

export default function App() {
  const [view, setView] = useState('start')
  const [sessionId, setSessionId] = useState(null)
  const [topic, setTopic] = useState('')
  const [history, setHistory] = useState([])

  function handleSessionStarted(id, seedContent, sessionTopic) {
    setSessionId(id)
    setTopic(sessionTopic)
    setHistory([{ role: 'assistant', content: seedContent }])
    setView('conversation')
  }

  function handleSessionEnded() {
    setView('summary')
  }

  function handleNewSession() {
    setSessionId(null)
    setTopic('')
    setHistory([])
    setView('start')
  }

  return (
    <div className="bg-gray-950 text-gray-100 min-h-screen flex flex-col">
      <nav className="border-b border-gray-800 bg-gray-950 sticky top-0 z-10">
        <div className="max-w-4xl mx-auto px-4 h-14 flex items-center justify-between">
          <span className="font-fraunces text-xl font-semibold text-amber-100 tracking-tight">Soltura</span>
          <button
            onClick={() => setView('vocab')}
            className="text-sm text-gray-400 hover:text-gray-200 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-800 font-mono"
          >
            Vocabulary
          </button>
        </div>
      </nav>

      {view === 'start' && (
        <StartView onSessionStarted={handleSessionStarted} />
      )}
      {view === 'conversation' && (
        <ConversationView
          sessionId={sessionId}
          topic={topic}
          history={history}
          onHistoryUpdate={setHistory}
          onEnd={handleSessionEnded}
        />
      )}
      {view === 'summary' && (
        <SummaryView sessionId={sessionId} onNewSession={handleNewSession} />
      )}
      {view === 'vocab' && (
        <VocabView onBack={() => setView(sessionId ? 'conversation' : 'start')} />
      )}
    </div>
  )
}
```

- [ ] **Step 2: Verify build still passes**

```bash
cd frontend && npm run build
```

---

## Task 6: StartView

**Files:**
- Create: `frontend/src/views/StartView.jsx`

- [ ] **Step 1: Create frontend/src/views/StartView.jsx**

```jsx
import { useState, useRef, useEffect } from 'react'
import LoadingBubble from '../components/LoadingBubble.jsx'

export default function StartView({ onSessionStarted }) {
  const [topic, setTopic] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const inputRef = useRef(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  async function startSession() {
    const trimmed = topic.trim()
    if (!trimmed) return
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/api/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ topic: trimmed }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || `Server error: ${res.status}`)
      }
      const data = await res.json()
      onSessionStarted(data.session_id, data.seed_content, trimmed)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter') startSession()
  }

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center px-4">
        <div className="w-full max-w-lg">
          <div className="text-center mb-12">
            <h1 className="font-fraunces text-6xl font-bold text-amber-100 mb-3 leading-tight">Soltura</h1>
            <p className="text-gray-400 text-lg font-mono">Your Spanish conversation partner</p>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-8 shadow-2xl">
            <LoadingBubble />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 flex items-center justify-center px-4">
      <div className="w-full max-w-lg">
        <div className="text-center mb-12">
          <h1 className="font-fraunces text-6xl font-bold text-amber-100 mb-3 leading-tight">Soltura</h1>
          <p className="text-gray-400 text-lg font-mono">Your Spanish conversation partner</p>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-xl p-8 shadow-2xl">
          <label className="block text-sm font-mono text-gray-400 mb-3 uppercase tracking-wider">
            What would you like to talk about?
          </label>
          <input
            ref={inputRef}
            type="text"
            value={topic}
            onChange={e => setTopic(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="e.g. My weekend plans, favourite films, cooking..."
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-gray-100 font-mono text-sm placeholder-gray-600 focus:outline-none focus:border-amber-600 focus:ring-1 focus:ring-amber-600 transition-colors mb-5"
          />
          {error && (
            <p className="text-red-400 font-mono text-sm mb-4 text-center">{error}</p>
          )}
          <button
            onClick={startSession}
            className="w-full text-white font-semibold py-3 px-6 rounded-lg transition-colors font-mono text-sm uppercase tracking-wider"
            style={{ backgroundColor: '#C1440E' }}
            onMouseOver={e => { e.currentTarget.style.backgroundColor = '#a33a0c' }}
            onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
          >
            Start Session
          </button>
        </div>

        <p className="text-center text-gray-600 text-xs font-mono mt-6">
          Press Enter to start · Cmd+Enter to send messages
        </p>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Verify build passes**

```bash
cd frontend && npm run build
```

---

## Task 7: ConversationView with SSE streaming

**Files:**
- Create: `frontend/src/views/ConversationView.jsx`

- [ ] **Step 1: Create frontend/src/views/ConversationView.jsx**

```jsx
import { useState, useRef, useEffect } from 'react'
import Message from '../components/Message.jsx'

export default function ConversationView({ sessionId, topic, history, onHistoryUpdate, onEnd }) {
  const [messages, setMessages] = useState(() =>
    history.map((h, i) => ({
      id: String(i),
      role: h.role,
      text: h.content,
      corrections: [],
      isLoading: false,
      isStreaming: false,
      isError: false,
    }))
  )
  const [inputText, setInputText] = useState('')
  const [submitDisabled, setSubmitDisabled] = useState(false)
  const [ending, setEnding] = useState(false)
  const threadRef = useRef(null)
  const inputRef = useRef(null)

  useEffect(() => {
    if (threadRef.current) {
      threadRef.current.scrollTop = threadRef.current.scrollHeight
    }
  }, [messages])

  async function submitTurn() {
    const text = inputText.trim()
    if (!text || !sessionId || submitDisabled) return

    const userMsgId = `${Date.now()}-user`
    const agentMsgId = `${Date.now()}-agent`

    setMessages(prev => [
      ...prev,
      { id: userMsgId, role: 'user', text, corrections: [], isLoading: false, isStreaming: false, isError: false },
      { id: agentMsgId, role: 'assistant', text: '', corrections: [], isLoading: true, isStreaming: false, isError: false },
    ])
    setInputText('')
    setSubmitDisabled(true)

    const updatedHistory = [...history, { role: 'user', content: text }]
    const historyToSend = updatedHistory.slice(-40)

    try {
      const response = await fetch(`/api/sessions/${sessionId}/turns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_text: text, history: historyToSend }),
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let fullText = ''
      let firstChunk = true

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop()

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const jsonStr = line.slice(6)
          if (jsonStr === '[DONE]') break
          try {
            const event = JSON.parse(jsonStr)

            if (event.type === 'chunk') {
              if (firstChunk) {
                firstChunk = false
                setMessages(prev => prev.map(m =>
                  m.id === agentMsgId ? { ...m, isLoading: false, isStreaming: true } : m
                ))
              }
              fullText += event.text
              setMessages(prev => prev.map(m =>
                m.id === agentMsgId ? { ...m, text: fullText } : m
              ))
            } else if (event.type === 'corrections') {
              setMessages(prev => prev.map(m =>
                m.id === agentMsgId ? { ...m, isStreaming: false, corrections: event.corrections } : m
              ))
            } else if (event.type === 'done') {
              onHistoryUpdate([...updatedHistory, { role: 'assistant', content: fullText }])
              setSubmitDisabled(false)
              inputRef.current?.focus()
            }
          } catch (_) { /* ignore malformed SSE lines */ }
        }
      }
    } catch (err) {
      setMessages(prev => prev.map(m =>
        m.id === agentMsgId
          ? { ...m, isLoading: false, isStreaming: false, isError: true, text: '[Error: could not get response. Please try again.]' }
          : m
      ))
      setSubmitDisabled(false)
      inputRef.current?.focus()
    }
  }

  async function endSession() {
    setEnding(true)
    try {
      await fetch(`/api/sessions/${sessionId}/end`, { method: 'POST' })
    } catch (_) { /* continue to summary even on error */ }
    onEnd()
  }

  return (
    <div className="flex-1 flex flex-col max-w-4xl w-full mx-auto px-4 py-4" style={{ height: 'calc(100vh - 3.5rem)' }}>
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <div className="flex items-center gap-3">
          <span className="text-xs font-mono text-gray-500 uppercase tracking-wider">Topic</span>
          <span className="font-fraunces text-amber-200 font-semibold text-lg">{topic}</span>
        </div>
        <button
          onClick={endSession}
          disabled={ending}
          className="text-sm font-mono text-gray-400 hover:text-red-400 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-900 border border-gray-800 hover:border-red-900 disabled:opacity-50"
        >
          {ending ? 'Ending…' : 'End Session'}
        </button>
      </div>

      <div
        ref={threadRef}
        className="flex-1 overflow-y-auto space-y-4 pr-1 pb-4 message-thread"
        style={{ minHeight: 0 }}
      >
        {messages.map(msg => (
          <Message key={msg.id} {...msg} />
        ))}
      </div>

      <div className="flex-shrink-0 mt-4 bg-gray-900 border border-gray-800 rounded-xl p-4">
        <textarea
          ref={inputRef}
          rows={3}
          value={inputText}
          onChange={e => setInputText(e.target.value)}
          onKeyDown={e => { if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') submitTurn() }}
          placeholder="Write in Spanish... (Cmd+Enter to send)"
          className="w-full bg-transparent text-gray-100 font-mono text-sm placeholder-gray-600 focus:outline-none resize-none leading-relaxed"
        />
        <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-800">
          <span className="text-xs font-mono text-gray-600">Cmd+Enter to send</span>
          <button
            onClick={submitTurn}
            disabled={submitDisabled}
            className="font-mono text-sm px-5 py-2 rounded-lg text-white font-medium transition-all disabled:opacity-40 disabled:cursor-not-allowed"
            style={{ backgroundColor: '#C1440E' }}
            onMouseOver={e => { if (!submitDisabled) e.currentTarget.style.backgroundColor = '#a33a0c' }}
            onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
          >
            Send →
          </button>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Verify build passes**

```bash
cd frontend && npm run build
```

---

## Task 8: SummaryView

**Files:**
- Create: `frontend/src/views/SummaryView.jsx`

- [ ] **Step 1: Create frontend/src/views/SummaryView.jsx**

```jsx
import { useState, useEffect } from 'react'
import { marked } from 'marked'

export default function SummaryView({ sessionId, onNewSession }) {
  const [data, setData] = useState(null)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch(`/api/sessions/${sessionId}/summary`)
      .then(r => { if (!r.ok) throw new Error(`Server error: ${r.status}`); return r.json() })
      .then(setData)
      .catch(err => setError(err.message))
  }, [sessionId])

  return (
    <div className="flex-1 flex items-start justify-center px-4 py-12">
      <div className="w-full max-w-2xl">
        <div className="text-center mb-8">
          <h2 className="font-fraunces text-4xl font-bold text-amber-100 mb-2">Session Complete</h2>
          <p className="text-gray-500 font-mono text-sm">Here's how you did</p>
        </div>

        {error && <p className="text-red-400 font-mono text-sm text-center mb-4">{error}</p>}

        <div className="grid grid-cols-2 gap-4 mb-8">
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-center">
            <div className="font-fraunces text-5xl font-bold text-amber-300 mb-2">{data?.turn_count ?? '—'}</div>
            <div className="text-gray-500 font-mono text-xs uppercase tracking-wider">Turns</div>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 text-center">
            <div className="font-fraunces text-5xl font-bold text-orange-400 mb-2">{data?.correction_count ?? '—'}</div>
            <div className="text-gray-500 font-mono text-xs uppercase tracking-wider">Corrections</div>
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-8">
          <h3 className="font-fraunces text-lg text-amber-200 mb-4 font-semibold">Feedback</h3>
          <div
            className="summary-text text-gray-300 font-mono text-sm leading-relaxed"
            dangerouslySetInnerHTML={{ __html: marked.parse(data?.summary || 'Loading…') }}
          />
        </div>

        <button
          onClick={onNewSession}
          className="w-full font-mono text-sm py-3 px-6 rounded-lg text-white font-semibold uppercase tracking-wider transition-colors"
          style={{ backgroundColor: '#C1440E' }}
          onMouseOver={e => { e.currentTarget.style.backgroundColor = '#a33a0c' }}
          onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
        >
          Start New Session
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Verify build passes**

```bash
cd frontend && npm run build
```

---

## Task 9: VocabView

**Files:**
- Create: `frontend/src/views/VocabView.jsx`

- [ ] **Step 1: Create frontend/src/views/VocabView.jsx**

```jsx
import { useState, useEffect } from 'react'

export default function VocabView({ onBack }) {
  const [entries, setEntries] = useState([])
  const [empty, setEmpty] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch('/api/vocab?limit=50')
      .then(r => { if (!r.ok) throw new Error(`Server error: ${r.status}`); return r.json() })
      .then(data => {
        const list = data.entries || []
        setEntries(list)
        setEmpty(list.length === 0)
      })
      .catch(err => setError(err.message))
  }, [])

  return (
    <div className="flex-1 flex flex-col max-w-4xl w-full mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6 flex-shrink-0">
        <h2 className="font-fraunces text-3xl font-bold text-amber-100">Vocabulary</h2>
        <button
          onClick={onBack}
          className="text-sm font-mono text-gray-400 hover:text-gray-200 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-800 border border-gray-800"
        >
          ← Back
        </button>
      </div>

      {error && <p className="text-red-400 font-mono text-sm text-center py-8">{error}</p>}

      <div className="flex-1 overflow-y-auto">
        <table className="w-full text-sm font-mono">
          <thead className="sticky top-0 bg-gray-950">
            <tr className="border-b border-gray-800">
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Original</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Corrected</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Category</th>
              <th className="text-right py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Times Seen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-900">
            {entries.map((entry, i) => (
              <tr key={i} className="vocab-row">
                <td className="py-3 px-4 text-gray-300">{entry.original}</td>
                <td className="py-3 px-4 text-green-400">{entry.corrected}</td>
                <td className="py-3 px-4">
                  <span className={`correction-badge category-${entry.category}`}>{entry.category}</span>
                </td>
                <td className="py-3 px-4 text-right text-gray-500">{entry.seen_count}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {empty && (
          <p className="text-center py-16 text-gray-600 font-mono text-sm">
            No corrections recorded yet. Start a session to build your vocabulary list.
          </p>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Do a full build and verify no errors**

```bash
cd frontend && npm run build
```

Expected: `web/index.html` and `web/assets/` generated with no TypeScript/JSX errors.

- [ ] **Step 3: Commit all components**

```bash
git add frontend/src/
git commit -m "feat: add React components — views, message, corrections, loading"
```

---

## Task 10: Update Makefile

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Replace Makefile with the following**

```makefile
OLLAMA_MODEL := gemma3:12b
BACKEND      := $(filter anthropic,$(MAKECMDGOALS))

.PHONY: install-model run anthropic build dev build-frontend

anthropic:
	@:

frontend/node_modules:
	cd frontend && npm install

## build-frontend: compile React source into web/
build-frontend: frontend/node_modules
	cd frontend && npm run build

## build: compile React + Go binary
build: build-frontend
	go build -o soltura .

## run [anthropic]: build frontend then start Go server (default: local Gemma)
##   make run             — use local Gemma model
##   make run anthropic   — use Anthropic API (requires ANTHROPIC_API_KEY)
run: build-frontend
	@if [ "$(BACKEND)" = "anthropic" ]; then \
		go run .; \
	else \
		LLM_BACKEND=ollama go run .; \
	fi

## dev [anthropic]: Vite hot-reload on :5173 + Go on :8080 (default: local Gemma)
##   make dev             — use local Gemma model
##   make dev anthropic   — use Anthropic API (requires ANTHROPIC_API_KEY)
dev: frontend/node_modules
	@if [ "$(BACKEND)" = "anthropic" ]; then \
		(trap 'kill %1 2>/dev/null' EXIT; (cd frontend && npm run dev) & go run .); \
	else \
		(trap 'kill %1 2>/dev/null' EXIT; (cd frontend && npm run dev) & LLM_BACKEND=ollama go run .); \
	fi

## install-model: install Ollama (via Homebrew) if absent, start server, then pull the model
install-model:
	@command -v ollama >/dev/null 2>&1 || { \
		echo "Ollama not found — installing via Homebrew..."; \
		brew install ollama; \
	}
	@curl -s http://localhost:11434/ >/dev/null 2>&1 || { \
		echo "Starting Ollama server..."; \
		ollama serve >/dev/null 2>&1 & \
		sleep 2; \
	}
	@echo "Pulling $(OLLAMA_MODEL) (this may take a while on first run)..."
	ollama pull $(OLLAMA_MODEL)
	@echo "Done. Run: make dev"
```

- [ ] **Step 2: Verify make build works end to end**

```bash
make build
```

Expected: frontend builds to `web/`, Go binary `soltura` compiled with no errors.

- [ ] **Step 3: Commit Makefile**

```bash
git add Makefile
git commit -m "chore: update Makefile — add dev/build-frontend, default to Gemma"
```

---

## Task 11: Clean up old files and update .gitignore

**Files:**
- Delete: `web/index.html`, `web/app.js`, `web/style.css`
- Modify: `.gitignore`

- [ ] **Step 1: Check current .gitignore**

```bash
cat .gitignore
```

- [ ] **Step 2: Add generated/dependency entries to .gitignore**

Add these lines to `.gitignore` (append, preserving existing content):

```
# Frontend build output (generated by Vite)
web/

# Node dependencies
frontend/node_modules/
```

- [ ] **Step 3: Remove old vanilla JS files from git tracking and disk**

```bash
git rm web/index.html web/app.js web/style.css
```

Expected: files removed from git index and disk.

- [ ] **Step 4: Rebuild web/ with Vite (now gitignored)**

```bash
make build-frontend
```

Expected: `web/index.html` and `web/assets/` created. These are gitignored — `git status` should not show them.

- [ ] **Step 5: Commit cleanup**

```bash
git add .gitignore
git commit -m "chore: remove vanilla JS files, gitignore web/ build output"
```

---

## Task 12: End-to-end verification

- [ ] **Step 1: Start the dev server with Gemma**

```bash
make dev
```

Expected: Vite starts on `:5173`, Go starts on `:8080`. Open `http://localhost:5173`.

- [ ] **Step 2: Verify start screen**

Open `http://localhost:5173`. Check:
- Soltura heading renders in Fraunces font
- Topic input focuses automatically
- Enter key triggers session start
- Loading bubble with timer appears while API call is in flight

- [ ] **Step 3: Verify conversation**

Start a session. Check:
- Opening Spanish question appears
- User messages appear right-aligned with orange border
- Agent replies stream in chunk by chunk with pulsing border
- Loading bubble with timer shows between sending and first token
- Corrections panel appears below agent message after streaming completes

- [ ] **Step 4: Verify session end + summary**

Click "End Session". Check:
- Summary view shows turn count and correction count
- Feedback text renders markdown correctly
- "Start New Session" returns to start screen

- [ ] **Step 5: Verify vocabulary view**

Click "Vocabulary" in nav. Check:
- Table shows corrections with category badges
- "← Back" returns to correct view

- [ ] **Step 6: Verify make run (production build)**

```bash
make run
```

Open `http://localhost:8080`. Verify all the same behaviour via Go's static file server.

- [ ] **Step 7: Verify make run anthropic works**

```bash
make run anthropic
```

Expected: Go starts without `LLM_BACKEND=ollama` — uses Anthropic API.
