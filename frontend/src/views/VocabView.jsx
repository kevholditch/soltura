import { useState, useEffect } from 'react'

function focusTitle(entry) {
  return `${entry.original} -> ${entry.corrected}`
}

function formatLastSeen(value) {
  const date = value ? new Date(value) : null
  if (!date || Number.isNaN(date.getTime())) return 'Unknown'

  const today = new Date()
  const todayStart = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  const dateStart = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const daysAgo = Math.floor((todayStart - dateStart) / 86400000)
  if (daysAgo === 0) return 'Today'
  if (daysAgo === 1) return 'Yesterday'
  if (daysAgo > 1 && daysAgo < 7) return `${daysAgo} days ago`

  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    year: date.getFullYear() === today.getFullYear() ? undefined : 'numeric',
  }).format(date)
}

function buildFocusTopics(entries) {
  const byKey = new Map()
  entries.filter(entry => !entry.learnt).forEach(entry => {
    const key = `${entry.original}\u0000${entry.corrected}\u0000${entry.category}`
    const existing = byKey.get(key)
    if (existing) {
      existing.entries.push(entry)
      existing.seenCount += entry.seen_count ?? 0
      const existingDate = new Date(existing.lastSeen)
      const nextDate = new Date(entry.last_seen)
      if (!Number.isNaN(nextDate.getTime()) && nextDate > existingDate) {
        existing.lastSeen = entry.last_seen
      }
      return
    }
    byKey.set(key, {
      id: key,
      title: focusTitle(entry),
      category: entry.category,
      seenCount: entry.seen_count ?? 0,
      lastSeen: entry.last_seen,
      entries: [entry],
    })
  })

  return Array.from(byKey.values()).sort((a, b) => {
    const dateDelta = new Date(b.lastSeen).getTime() - new Date(a.lastSeen).getTime()
    if (dateDelta !== 0) return dateDelta
    return b.seenCount - a.seenCount
  })
}

export default function VocabView({ onBack, onDrillStart }) {
  const [entries, setEntries] = useState([])
  const [empty, setEmpty] = useState(false)
  const [error, setError] = useState('')
  const [selectedTopic, setSelectedTopic] = useState(null)

  useEffect(() => {
    fetch('/api/vocab?limit=50&sort=recent')
      .then(r => { if (!r.ok) throw new Error(`Server error: ${r.status}`); return r.json() })
      .then(data => {
        const list = data.entries || []
        setEntries(list)
        setEmpty(list.length === 0)
        setSelectedTopic(current => current && list.some(entry => current.entries.some(topicEntry => topicEntry.id === entry.id)) ? current : null)
      })
      .catch(err => setError(err.message))
  }, [])

  const focusTopics = buildFocusTopics(entries)
  const visibleEvidence = selectedTopic?.entries ?? []

  return (
    <div className="flex-1 flex flex-col max-w-4xl w-full mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6 flex-shrink-0">
        <h2 className="font-fraunces text-3xl font-bold text-amber-800 dark:text-amber-100">Vocabulary</h2>
        <button
          onClick={onBack}
          className="text-sm font-mono text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 border border-gray-200 dark:border-gray-800"
        >
          ← Back
        </button>
      </div>

      {error && <p className="text-red-600 dark:text-red-400 font-mono text-sm text-center py-8">{error}</p>}

      {!empty && focusTopics.length > 0 && (
        <div className="grid md:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)] gap-5 mb-8">
          <section aria-labelledby="focus-next-heading">
            <div className="flex items-baseline justify-between border-b border-gray-200 dark:border-gray-800 pb-2 mb-3">
              <h3 id="focus-next-heading" className="font-fraunces text-xl font-semibold text-amber-800 dark:text-amber-100">Focus next</h3>
              <span className="font-mono text-xs text-gray-500 dark:text-gray-400">{focusTopics.length} topics</span>
            </div>
            <div className="space-y-2">
              {focusTopics.slice(0, 6).map(topic => (
                <button
                  key={topic.id}
                  type="button"
                  onClick={() => setSelectedTopic(topic)}
                  className={`w-full text-left border rounded-lg p-3 transition-colors focus:outline-none focus:ring-2 focus:ring-amber-500 ${
                    selectedTopic?.id === topic.id
                      ? 'border-amber-400 dark:border-amber-700 bg-amber-50 dark:bg-amber-950/20'
                      : 'border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-amber-300 dark:hover:border-amber-800'
                  }`}
                  aria-label={topic.title}
                >
                  <span className="block font-mono text-sm text-gray-900 dark:text-gray-100">{topic.title}</span>
                  <span className="block font-mono text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Seen {topic.seenCount} {topic.seenCount === 1 ? 'time' : 'times'} · last seen {formatLastSeen(topic.lastSeen)}
                  </span>
                </button>
              ))}
            </div>
          </section>

          <section aria-labelledby="evidence-heading">
            <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-4 bg-white dark:bg-gray-900 min-h-[14rem]">
              {selectedTopic ? (
                <>
                  <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-4">
                    <div>
                      <h3 id="evidence-heading" className="font-fraunces text-xl font-semibold text-amber-800 dark:text-amber-100">Evidence</h3>
                      <p className="font-mono text-sm text-gray-700 dark:text-gray-300 mt-1">{selectedTopic.title}</p>
                      <p className="font-mono text-xs text-gray-500 dark:text-gray-400 mt-1">
                        {selectedTopic.category} · Seen {selectedTopic.seenCount} {selectedTopic.seenCount === 1 ? 'time' : 'times'} · Last seen {formatLastSeen(selectedTopic.lastSeen)}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => onDrillStart?.(visibleEvidence.map(entry => entry.id))}
                      className="font-mono text-sm px-4 py-2 rounded-lg text-white font-medium bg-orange-700 hover:bg-orange-800 disabled:opacity-50"
                      disabled={!onDrillStart}
                    >
                      Start drill
                    </button>
                  </div>
                  <div className="space-y-3">
                    {visibleEvidence.map(entry => (
                      <div key={entry.id} className="border-t border-gray-100 dark:border-gray-800 pt-3">
                        <div className="correction-row">
                          <span className="correction-original">{entry.original}</span>
                          <span className="correction-arrow">→</span>
                          <span className="correction-corrected">{entry.corrected}</span>
                          <span className={`correction-badge category-${entry.category}`}>{entry.category}</span>
                        </div>
                        <p className="correction-explanation">{entry.explanation}</p>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="h-full min-h-[12rem] flex flex-col justify-center">
                  <h3 id="evidence-heading" className="font-fraunces text-xl font-semibold text-amber-800 dark:text-amber-100 mb-2">Evidence</h3>
                  <p className="font-mono text-sm text-gray-500 dark:text-gray-400">Choose a focus topic to review examples before drilling.</p>
                </div>
              )}
            </div>
          </section>
        </div>
      )}

      <div className="flex-1 overflow-y-auto">
        <table className="w-full text-sm font-mono">
          <thead className="sticky top-0 bg-white dark:bg-gray-950">
            <tr className="border-b border-gray-200 dark:border-gray-800">
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Original</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Corrected</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Category</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Last seen</th>
              <th className="text-right py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Times Seen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 dark:divide-gray-900">
            {entries.map((entry, i) => (
              <tr key={i} className="vocab-row">
                <td className="py-3 px-4 text-gray-700 dark:text-gray-300">{entry.original}</td>
                <td className="py-3 px-4 text-green-700 dark:text-green-400">{entry.corrected}</td>
                <td className="py-3 px-4">
                  <span className={`correction-badge category-${entry.category}`}>{entry.category}</span>
                </td>
                <td className="py-3 px-4 text-gray-500">{formatLastSeen(entry.last_seen)}</td>
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
