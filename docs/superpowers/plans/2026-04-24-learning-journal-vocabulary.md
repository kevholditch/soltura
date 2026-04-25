# Learning Journal and Vocabulary Insights Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a review-first learning journal, improve vocabulary analysis, and allow focused drills from vocabulary evidence.

**Architecture:** Add backend session list/review APIs over existing SQLite data, extend vocabulary sorting/fields, and allow `POST /api/drills/start` to accept optional scoped vocab IDs. Add React journal/session-review surfaces and update vocabulary to expose last-seen analysis and focus-topic evidence.

**Tech Stack:** Go + Chi + SQLite backend, React 18 + Vite frontend, Playwright e2e with `LLM_BACKEND=test`.

---

## File Map

### Backend

- Modify `models/models.go`
  - Add `SessionListItem` and `SessionReview` response models.
- Modify `store/sqlite.go`
  - Add session listing/review methods.
  - Add vocab lookup by IDs.
  - Add sort-aware vocab listing.
  - Scan learnt fields in vocab responses.
- Modify `handlers/session.go`
  - Add `List` and `Review` handlers.
- Modify `handlers/vocabulary.go`
  - Read `sort` query param and return richer vocab fields.
- Modify `handlers/drill.go`
  - Parse optional `{ "vocab_ids": [] }` in `Start`.
- Modify `main.go`
  - Register `GET /api/sessions` and `GET /api/sessions/{sessionID}/review`.

### Frontend

- Modify `frontend/src/App.jsx`
  - Add journal/session review view states.
- Modify `frontend/src/components/MenuBar.jsx`
  - Add Journal button.
- Create `frontend/src/views/JournalView.jsx`
  - Dashboard with grouped sessions and focus topics.
- Create `frontend/src/views/SessionReviewView.jsx`
  - Read-only transcript/corrections/summary review.
- Modify `frontend/src/views/VocabView.jsx`
  - Add last-seen column, recent grouping, focus topic evidence.
- Modify `frontend/src/views/DrillView.jsx`
  - Accept optional initial scoped vocab IDs.
- Modify `frontend/src/views/StartView.jsx`
  - Shorten drills copy.

### Tests

- Create `frontend/e2e/journal_flow.spec.js`
- Modify `frontend/e2e/vocab_flow.spec.js`
- Modify `frontend/e2e/drills_flow.spec.js`

---

## Task 1: Write Failing Playwright Tests First

**Files:**
- Create: `frontend/e2e/journal_flow.spec.js`
- Modify: `frontend/e2e/vocab_flow.spec.js`
- Modify: `frontend/e2e/drills_flow.spec.js`

- [ ] **Step 1: Add journal e2e spec**

Create `frontend/e2e/journal_flow.spec.js`:

```js
import { expect, test } from '@playwright/test'
import { completeText, correctionArray, streamText } from './helpers/fixture.js'

async function completeSession(page, topic, userText) {
  await page.getByLabel('New chat session').click()
  await page.getByPlaceholder(/My weekend plans/i).fill(topic)
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill(userText)
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText(streamText('conversation_stream'))).toBeVisible()

  await page.getByRole('button', { name: /End Session/i }).click()
  await expect(page.getByRole('heading', { name: 'Session Complete' })).toBeVisible()
}

test('journal lists old sessions and opens read-only review', async ({ page }) => {
  const correction = correctionArray()[0]

  await page.goto('/')
  await completeSession(page, 'Weekend plans', 'Voy a ir a el skatepark')

  await page.getByLabel('Journal').click()

  await expect(page.getByRole('heading', { name: 'Journal' })).toBeVisible()
  await expect(page.getByText('Today')).toBeVisible()
  await expect(page.getByText('Weekend plans')).toBeVisible()
  await expect(page.getByText(/1 correction/i)).toBeVisible()

  await page.getByText('Weekend plans').click()

  await expect(page.getByRole('heading', { name: 'Weekend plans' })).toBeVisible()
  await expect(page.getByText('Voy a ir a el skatepark')).toBeVisible()
  await expect(page.getByText(streamText('conversation_stream'))).toBeVisible()
  await expect(page.getByText(correction.original)).toBeVisible()
  await expect(page.getByText(correction.corrected)).toBeVisible()
  await expect(page.getByText(/What went well/i)).toBeVisible()
  await expect(page.getByPlaceholder(/Write in Spanish/i)).toHaveCount(0)
  await expect(page.getByRole('button', { name: /Start new chat on this topic/i })).toBeVisible()
})
```

- [ ] **Step 2: Extend vocabulary spec for last seen and evidence**

Update `frontend/e2e/vocab_flow.spec.js` with a second test:

```js
test('vocabulary shows last seen and focus topic evidence before drilling', async ({ page }) => {
  const correction = correctionArray()[0]

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill('Spanish chat')
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill('Voy a ir a el skatepark')
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText(completeText('conversation_stream'))).toBeVisible()

  await page.getByLabel('Vocabulary').click()

  await expect(page.getByRole('columnheader', { name: /Last seen/i })).toBeVisible()
  await expect(page.getByText(/Today/i)).toBeVisible()
  await expect(page.getByText(`${correction.original} -> ${correction.corrected}`)).toBeVisible()

  await page.getByText(`${correction.original} -> ${correction.corrected}`).click()

  await expect(page.getByRole('heading', { name: /Evidence/i })).toBeVisible()
  await expect(page.getByText(correction.explanation)).toBeVisible()
  await expect(page.getByRole('button', { name: /Start drill/i })).toBeVisible()

  await page.getByRole('button', { name: /Start drill/i }).click()
  await expect(page.getByText(/Completa: Voy ___ skatepark/i)).toBeVisible()
})
```

- [ ] **Step 3: Update drills copy spec**

Add to `frontend/e2e/drills_flow.spec.js`:

```js
test('drills menu uses concise copy', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: 'Drills' }).click()

  await expect(page.getByText(/Practice your common mistakes/i)).toBeVisible()
  await expect(page.getByText(/I'll analyse your most common mistakes/i)).toHaveCount(0)
})
```

- [ ] **Step 4: Run focused e2e tests and confirm they fail**

Run:

```bash
cd frontend && npm run build && PORT=8092 npm run test:e2e -- --grep "journal|vocabulary shows last seen|drills menu uses concise copy"
```

Expected: tests fail because Journal navigation/view, last-seen focus evidence, scoped drill start, and copy changes are not implemented yet.

---

## Task 2: Add Backend Session Journal APIs

**Files:**
- Modify: `models/models.go`
- Modify: `store/sqlite.go`
- Modify: `handlers/session.go`
- Modify: `main.go`

- [ ] **Step 1: Add response models**

Add to `models/models.go`:

```go
type SessionListItem struct {
	ID              string    `json:"id"`
	Topic           string    `json:"topic"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	TurnCount       int       `json:"turn_count"`
	CorrectionCount int       `json:"correction_count"`
	Categories      []string  `json:"categories"`
}

type SessionReview struct {
	Session         Session `json:"session"`
	Turns           []Turn  `json:"turns"`
	CorrectionCount int     `json:"correction_count"`
	Categories      []string `json:"categories"`
}
```

- [ ] **Step 2: Extend store interface**

Add to `Store` in `store/sqlite.go`:

```go
ListSessions(limit int) ([]models.SessionListItem, error)
GetSessionReview(sessionID string) (*models.SessionReview, error)
GetVocabByIDs(ids []string) ([]models.VocabEntry, error)
```

- [ ] **Step 3: Implement `ListSessions`**

Add a query ordered by newest first. Aggregate turn/correction counts and categories in Go so the SQL stays portable.

```go
func (s *SQLiteStore) ListSessions(limit int) ([]models.SessionListItem, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, topic, started_at, ended_at FROM sessions ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var items []models.SessionListItem
	for rows.Next() {
		sess, err := scanSessionRows(rows)
		if err != nil {
			return nil, err
		}
		turns, err := s.GetTurns(sess.ID)
		if err != nil {
			return nil, err
		}
		corrections, err := s.GetCorrections(sess.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, models.SessionListItem{
			ID: sess.ID, Topic: sess.Topic, StartedAt: sess.StartedAt, EndedAt: sess.EndedAt,
			TurnCount: len(turns), CorrectionCount: len(corrections), Categories: uniqueCorrectionCategories(corrections),
		})
	}
	return items, rows.Err()
}
```

Add helpers `scanSessionRows` and `uniqueCorrectionCategories` next to existing scan helpers.

- [ ] **Step 4: Implement `GetSessionReview`**

```go
func (s *SQLiteStore) GetSessionReview(sessionID string) (*models.SessionReview, error) {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	turns, err := s.GetTurns(sessionID)
	if err != nil {
		return nil, err
	}
	corrections, err := s.GetCorrections(sessionID)
	if err != nil {
		return nil, err
	}
	return &models.SessionReview{
		Session: *session, Turns: turns, CorrectionCount: len(corrections), Categories: uniqueCorrectionCategories(corrections),
	}, nil
}
```

- [ ] **Step 5: Add handlers**

Add methods to `handlers/session.go`:

```go
func (s *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	sessions, err := s.store.ListSessions(limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sessions": sessions})
}

func (s *SessionHandler) Review(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	review, err := s.store.GetSessionReview(sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}
```

- [ ] **Step 6: Register routes**

In `main.go`:

```go
r.Get("/api/sessions", sessionHandler.List)
r.Get("/api/sessions/{sessionID}/review", sessionHandler.Review)
```

- [ ] **Step 7: Run Go tests**

Run:

```bash
make test-go
```

Expected: pass.

---

## Task 3: Add Journal and Session Review Frontend

**Files:**
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/components/MenuBar.jsx`
- Create: `frontend/src/views/JournalView.jsx`
- Create: `frontend/src/views/SessionReviewView.jsx`

- [ ] **Step 1: Add Journal navigation**

In `MenuBar.jsx`, import a suitable lucide icon such as `History` and add:

```jsx
<button
  onClick={onJournalOpen}
  aria-label="Journal"
  className={menuButtonClass(activeView === 'journal' || activeView === 'sessionReview')}
>
  <History size={16} />
  <span className="hidden md:inline">Journal</span>
</button>
```

- [ ] **Step 2: Wire App view state**

In `App.jsx`, add state:

```jsx
const [reviewSessionId, setReviewSessionId] = useState(null)
```

Add handlers:

```jsx
function handleJournalOpen() {
  setView('journal')
}

function handleSessionReviewOpen(id) {
  setReviewSessionId(id)
  setView('sessionReview')
}

function handleStartNewChatOnTopic(nextTopic) {
  handleTopicSelected(nextTopic)
}
```

Render:

```jsx
{view === 'journal' && (
  <JournalView onSessionOpen={handleSessionReviewOpen} onDrillStart={(ids) => { setScopedDrillVocabIds(ids); setView('drills') }} />
)}
{view === 'sessionReview' && (
  <SessionReviewView sessionId={reviewSessionId} onBack={handleJournalOpen} onStartNewChat={handleStartNewChatOnTopic} />
)}
```

- [ ] **Step 3: Create `JournalView.jsx`**

Implement fetch from `/api/sessions?limit=50`, group sessions by local date bucket, render heading "Journal", date group headers, and clickable session rows.

- [ ] **Step 4: Create `SessionReviewView.jsx`**

Fetch:

- `/api/sessions/${sessionId}/review`
- `/api/sessions/${sessionId}/summary`

Render the stored turns read-only using the existing `Message` component. Render corrections and summary below. Do not render the chat textarea.

- [ ] **Step 5: Run journal spec**

Run:

```bash
cd frontend && npm run build && PORT=8092 npm run test:e2e -- --grep "journal lists old sessions"
```

Expected: pass after Tasks 2 and 3.

---

## Task 4: Improve Vocabulary Analysis and Focus Evidence

**Files:**
- Modify: `store/sqlite.go`
- Modify: `handlers/vocabulary.go`
- Modify: `frontend/src/views/VocabView.jsx`

- [ ] **Step 1: Add sort-aware vocab query**

Change store vocab listing to accept a sort option or add `GetVocabSorted(limit int, sort string)`. For `recent`, order by:

```sql
ORDER BY last_seen DESC, seen_count DESC
```

For `frequency`, preserve:

```sql
ORDER BY seen_count DESC, last_seen DESC
```

Scan `learnt` and `learnt_at` as well as existing fields.

- [ ] **Step 2: Update vocabulary handler**

Read:

```go
sort := r.URL.Query().Get("sort")
if sort == "" {
	sort = "recent"
}
```

Return entries with last seen and learnt fields.

- [ ] **Step 3: Update `VocabView.jsx`**

Fetch `/api/vocab?limit=50&sort=recent`.

Render:

- Focus topic list above or beside the table.
- Evidence panel when a topic is selected.
- Table columns: Original, Corrected, Category, Last seen, Times seen.

Build focus topics on the frontend from entries:

```js
function focusTitle(entry) {
  return `${entry.original} -> ${entry.corrected}`
}
```

Use matching `original`, `corrected`, and `category` to group evidence rows.

- [ ] **Step 4: Run vocabulary spec**

Run:

```bash
cd frontend && npm run build && PORT=8092 npm run test:e2e -- --grep "vocabulary shows last seen"
```

Expected: pass after Task 4 and Task 5's scoped drill handoff.

---

## Task 5: Add Scoped Drill Start From Evidence

**Files:**
- Modify: `store/sqlite.go`
- Modify: `handlers/drill.go`
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/views/DrillView.jsx`
- Modify: `frontend/src/views/VocabView.jsx`

- [ ] **Step 1: Implement `GetVocabByIDs`**

Fetch only unlearnt vocab entries whose IDs are in the request. Preserve the selected order if practical; otherwise order by `seen_count DESC, last_seen DESC`.

- [ ] **Step 2: Parse optional scoped IDs in drill start**

At the top of `DrillHandler.Start`, decode an optional body:

```go
var body struct {
	VocabIDs []string `json:"vocab_ids"`
}
_ = json.NewDecoder(r.Body).Decode(&body)
```

If `body.VocabIDs` has entries, call `GetVocabByIDs`; otherwise call `GetUnlearntVocab`.

- [ ] **Step 3: Let App pass scoped IDs into DrillView**

Add:

```jsx
const [scopedDrillVocabIds, setScopedDrillVocabIds] = useState([])
```

When starting a top-level drill, clear scope:

```jsx
function handleDrillsStart() {
  setScopedDrillVocabIds([])
  setView('drills')
}
```

When starting from evidence, set the selected IDs and open drills.

- [ ] **Step 4: Update `DrillView` start request**

Accept prop:

```jsx
export default function DrillView({ onExit, initialVocabIds = [] }) {
```

Post JSON body when scoped:

```js
const body = initialVocabIds.length > 0 ? JSON.stringify({ vocab_ids: initialVocabIds }) : undefined
const res = await fetch('/api/drills/start', {
  method: 'POST',
  headers: body ? { 'Content-Type': 'application/json' } : undefined,
  body,
})
```

- [ ] **Step 5: Connect evidence Start drill button**

In `VocabView.jsx`, call the provided `onDrillStart(vocabIds)` prop from the evidence panel primary button.

- [ ] **Step 6: Run focused drill-from-vocab test**

Run:

```bash
cd frontend && npm run build && PORT=8092 npm run test:e2e -- --grep "focus topic evidence"
```

Expected: pass.

---

## Task 6: Shorten Drills Menu Copy

**Files:**
- Modify: `frontend/src/views/StartView.jsx`

- [ ] **Step 1: Replace verbose copy**

Change the drills panel paragraph to:

```jsx
<p className="text-gray-600 dark:text-gray-400 font-mono text-sm mb-6 leading-relaxed">
  Practice your common mistakes.
</p>
```

- [ ] **Step 2: Run drills copy spec**

Run:

```bash
cd frontend && npm run build && PORT=8092 npm run test:e2e -- --grep "drills menu uses concise copy"
```

Expected: pass.

---

## Task 7: Full Verification

**Files:**
- No new files.

- [ ] **Step 1: Run Go tests**

```bash
make test-go
```

Expected: pass.

- [ ] **Step 2: Run full Playwright suite in test mode**

```bash
PORT=8092 make test-e2e
```

Expected: pass. Playwright config starts the app with `LLM_BACKEND=test`, a temporary SQLite database, and `testdata/llm/default.json`.

- [ ] **Step 3: Run full test target**

```bash
PORT=8091 make test-all
```

Expected: pass.

- [ ] **Step 4: Manual browser smoke check**

Start the app through the normal dev target:

```bash
make dev
```

Confirm:

- Journal opens from nav.
- Old session review is read-only.
- Vocabulary table shows Last seen.
- Focus topic opens evidence before drill.
- Start drill from evidence reaches the existing drill interface.

---

## Self-Review Notes

- Spec coverage: journal, read-only review, vocabulary last seen, evidence-first focus topics, scoped drill start, copy change, and test-mode Playwright coverage are all represented.
- Scope: no chat resume, no LLM-generated analytics, no account/sync/search.
- TDD path: Task 1 creates failing Playwright tests first, then backend/frontend tasks make them pass.
