# Learning Journal and Vocabulary Insights

**Date:** 2026-04-24  
**Status:** Draft for review

## Context

Soltura already records the data needed for a learning journal: sessions, turns, corrections, vocabulary entries, `seen_count`, and `last_seen`. The current UI only exposes the active chat, the post-session summary, a flat vocabulary table, and top-level drills. Users cannot review old chats, and vocabulary is hard to analyse because it is presented as a raw list sorted primarily by frequency.

This design turns the stored learning data into a journal/dashboard. The journal is review-first: old sessions are evidence of learning progress, not resumable chat threads.

## Goals

- Add a learning journal where users can review old chat sessions.
- Make old chats feel like a learning dashboard/journal, not a generic chat archive.
- Keep old sessions read-only for the first version.
- Improve vocabulary analysis with `last_seen`, recent ordering, grouping, and focus topics.
- Let users inspect examples/evidence for a focus topic before starting a drill.
- Start scoped drills from a selected focus topic.
- Shorten the drills menu copy to something like "Practice your common mistakes."
- Drive implementation with new Playwright tests running through the existing deterministic test LLM backend.

## Non-Goals

- Do not support resuming or mutating an old chat in this version.
- Do not add LLM-generated weekly analytics yet.
- Do not add account-level sync, search across all transcripts, or manual editing of vocabulary.
- Do not replace the existing top-level Drills flow; add a focused entry path alongside it.

## Product Direction

The chosen direction is a hybrid journal:

- The main area is a dated session journal.
- An insight area shows "Focus next" topics derived from recent, repeated vocabulary corrections.
- Sessions and vocabulary entries reinforce each other: a session explains what happened; a focus topic explains what to practise next.

Date headers such as "Today" and "This week" are grouping and orientation elements. They show totals and may become collapsible later, but they are not the primary click target. Session rows open old session review.

## Navigation

Add a top-level "Journal" entry to the app navigation.

Recommended menu order:

- New Chat
- Journal
- Drills
- Vocabulary
- Theme toggle

The existing Vocabulary button can remain because it still serves users who want the full data table. The Journal becomes the more guided dashboard entry point.

## Journal Dashboard

The Journal view has two regions:

1. Session timeline
2. Focus next panel

### Session Timeline

Sessions are grouped by relative date buckets:

- Today
- This week
- Older

Each group header shows aggregate counts:

- Number of sessions
- Number of corrections

Each session row shows:

- Topic
- Local date/time
- Turn count
- Correction count
- Top correction categories

Clicking a session row opens a read-only session review.

### Session Review

The review screen shows:

- Session topic and date
- Read-only transcript from stored turns
- Corrections from that session
- Existing generated summary for that session

The review should not show a message composer. It should offer "Start new chat on this topic" as the safe alternative to resuming the old thread.

## Vocabulary Insights

The Vocabulary view should become useful for analysis rather than just inspection.

Changes:

- Add a visible "Last seen" column.
- Default ordering should be `last_seen DESC`, with `seen_count DESC` as a secondary sort.
- Keep "Times seen" because frequency still matters.
- Surface learnt/unlearnt state where available.
- Group or label entries by recency: Today, This week, Older.
- Add focus topics, ordered by recent repeated mistakes.

For the first implementation, a focus topic is derived deterministically from one or more vocab entries. The initial grouping can be by correction pair and category:

- `original`
- `corrected`
- `category`

The displayed focus title can be the correction pair, such as `a el -> al`. This is reliable with current data and avoids introducing a new LLM analytics pass. Later, an LLM can group related corrections into richer topic labels like `por / para`.

## Focus Topic Evidence

Clicking a focus topic should first show examples/evidence, not immediately start a drill.

The evidence panel shows:

- Topic title
- Seen count
- Last seen date
- Category
- Example correction rows with original, corrected, and explanation
- Primary button: "Start drill"
- Secondary option: return/close

This gives the learner context before practice and makes drills feel grounded in their own mistakes.

## Focused Drills

Starting a drill from a focus topic should scope the drill to the selected vocab IDs.

The existing `/api/drills/start` behavior remains the fallback when no scope is provided. When scoped IDs are provided, the drill start prompt receives only those vocab entries, so the generated drill targets the selected topic.

## API Shape

Add session review endpoints:

- `GET /api/sessions?limit=50`
  - Returns journal session summaries ordered by `started_at DESC`.
- `GET /api/sessions/{sessionID}/review`
  - Returns session metadata, stored turns with corrections, and aggregate counts.

Extend vocabulary:

- `GET /api/vocab?limit=50&sort=recent`
  - Supports `recent` and `frequency`.
  - Returns `last_seen`, `learnt`, and `learnt_at`.

Extend drill start:

- `POST /api/drills/start`
  - Accepts optional JSON body: `{ "vocab_ids": ["..."] }`.
  - If IDs are present, fetch only those unlearnt vocab entries.
  - If absent, keep existing top-level drill behavior.

## Frontend Components

Add:

- `JournalView.jsx`
- `SessionReviewView.jsx`
- Small helpers for date grouping and formatting

Modify:

- `App.jsx` to add `journal` and `sessionReview` views.
- `MenuBar.jsx` to add Journal navigation.
- `VocabView.jsx` to add last seen, recency ordering, focus topics, and evidence panel.
- `DrillView.jsx` to accept optional initial vocab IDs.
- `StartView.jsx` to shorten drills copy.

## Testing Strategy

Implementation should be driven by Playwright first. The existing Playwright config already starts the Go server with:

- `LLM_BACKEND=test`
- isolated SQLite database in `/tmp`
- deterministic fixture file

Add e2e coverage for:

- Journal lists previous sessions grouped by date.
- Clicking a session row opens read-only review.
- Session review shows stored transcript, corrections, and summary.
- Vocabulary shows `Last seen` and orders recent items first.
- Clicking a focus topic shows evidence before drill.
- Starting a drill from evidence opens a scoped drill.
- Drills menu copy is concise.

Run focused specs during implementation, then the full suite:

```bash
PORT=8092 make test-e2e
make test-go
```

## Open Decisions

- Exact visual density of the Journal can be tuned during implementation.
- Whether date groups should be collapsible can wait.
- Whether the Journal eventually replaces the separate Vocabulary page can wait.

