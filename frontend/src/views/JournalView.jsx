import { useEffect, useMemo, useState } from 'react'
import { BookOpen, CalendarDays } from 'lucide-react'

function parseDate(value) {
  const date = value ? new Date(value) : null
  return date && !Number.isNaN(date.getTime()) ? date : null
}

function startOfLocalDay(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function bucketForSession(session, now = new Date()) {
  const startedAt = parseDate(session.started_at)
  if (!startedAt) return 'Older'

  const today = startOfLocalDay(now)
  const sessionDay = startOfLocalDay(startedAt)
  const daysAgo = Math.floor((today - sessionDay) / 86400000)

  if (daysAgo === 0) return 'Today'
  if (daysAgo > 0 && daysAgo < 7) return 'This week'
  return 'Older'
}

function formatSessionDate(value) {
  const date = parseDate(value)
  if (!date) return 'Date unavailable'

  const sameYear = date.getFullYear() === new Date().getFullYear()
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    year: sameYear ? undefined : 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(date)
}

function pluralize(count, singular) {
  return `${count} ${singular}${count === 1 ? '' : 's'}`
}

function groupSessions(sessions) {
  const groups = [
    { label: 'Today', sessions: [] },
    { label: 'This week', sessions: [] },
    { label: 'Older', sessions: [] },
  ]
  const byLabel = Object.fromEntries(groups.map(group => [group.label, group]))
  sessions.forEach(session => {
    byLabel[bucketForSession(session)].sessions.push(session)
  })
  return groups.filter(group => group.sessions.length > 0)
}

export default function JournalView({ onSessionOpen }) {
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')

    fetch('/api/sessions?limit=50')
      .then(response => {
        if (!response.ok) throw new Error(`Server error: ${response.status}`)
        return response.json()
      })
      .then(data => {
        if (cancelled) return
        setSessions(Array.isArray(data?.sessions) ? data.sessions : [])
      })
      .catch(err => {
        if (cancelled) return
        setError(err.message)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => { cancelled = true }
  }, [])

  const groupedSessions = useMemo(() => groupSessions(sessions), [sessions])

  return (
    <main className="flex-1 w-full max-w-4xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between gap-4 mb-8">
        <div>
          <h2 className="font-fraunces text-4xl font-bold text-amber-800 dark:text-amber-100">Journal</h2>
          <p className="text-gray-500 dark:text-gray-400 font-mono text-sm mt-1">Recent completed and in-progress sessions</p>
        </div>
        <CalendarDays className="text-amber-700 dark:text-amber-300 hidden sm:block" size={28} aria-hidden="true" />
      </div>

      {loading && (
        <p className="text-gray-500 dark:text-gray-400 font-mono text-sm">Loading sessions...</p>
      )}

      {error && (
        <div className="border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/30 rounded-lg p-4">
          <p className="text-red-700 dark:text-red-300 font-mono text-sm">Could not load journal: {error}</p>
        </div>
      )}

      {!loading && !error && sessions.length === 0 && (
        <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-8 text-center">
          <BookOpen className="mx-auto text-gray-300 dark:text-gray-700 mb-3" size={32} aria-hidden="true" />
          <p className="font-fraunces text-xl text-gray-800 dark:text-gray-200 mb-1">No sessions yet</p>
          <p className="text-gray-500 dark:text-gray-400 font-mono text-sm">Finished conversations will appear here for review.</p>
        </div>
      )}

      {!loading && !error && groupedSessions.length > 0 && (
        <div className="space-y-8">
          {groupedSessions.map(group => (
            <section key={group.label} aria-labelledby={`journal-${group.label.replace(/\s+/g, '-').toLowerCase()}`}>
              <div className="flex items-baseline justify-between border-b border-gray-200 dark:border-gray-800 pb-2 mb-3">
                <h3 id={`journal-${group.label.replace(/\s+/g, '-').toLowerCase()}`} className="font-fraunces text-2xl font-semibold text-amber-800 dark:text-amber-100">
                  {group.label}
                </h3>
                <span className="font-mono text-xs text-gray-500 dark:text-gray-400">{pluralize(group.sessions.length, 'session')}</span>
              </div>
              <div className="space-y-2">
                {group.sessions.map(session => (
                  <button
                    key={session.id}
                    type="button"
                    onClick={() => onSessionOpen(session.id)}
                    className="w-full text-left border border-gray-200 dark:border-gray-800 rounded-lg p-4 bg-white dark:bg-gray-900 hover:border-amber-300 dark:hover:border-amber-700 hover:bg-amber-50/50 dark:hover:bg-amber-950/20 transition-colors focus:outline-none focus:ring-2 focus:ring-amber-500"
                    aria-label={`${session.topic}, ${pluralize(session.turn_count ?? 0, 'turn')}, ${pluralize(session.correction_count ?? 0, 'correction')}`}
                  >
                    <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-2">
                      <div>
                        <p className="font-fraunces text-lg font-semibold text-gray-900 dark:text-gray-100">{session.topic || 'Untitled session'}</p>
                        <p className="font-mono text-xs text-gray-500 dark:text-gray-400 mt-1">{formatSessionDate(session.started_at)}</p>
                      </div>
                      <div className="font-mono text-xs text-gray-500 dark:text-gray-400 sm:text-right">
                        <span>{pluralize(session.turn_count ?? 0, 'turn')}</span>
                        <span className="mx-2 text-gray-300 dark:text-gray-700">/</span>
                        <span>{pluralize(session.correction_count ?? 0, 'correction')}</span>
                      </div>
                    </div>
                    {(session.categories ?? []).length > 0 && (
                      <div className="flex flex-wrap gap-2 mt-3">
                        {session.categories.map(category => (
                          <span key={category} className="font-mono text-[11px] uppercase tracking-wider px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300">
                            {category}
                          </span>
                        ))}
                      </div>
                    )}
                  </button>
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </main>
  )
}
