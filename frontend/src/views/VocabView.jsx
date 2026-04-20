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
        <h2 className="font-fraunces text-3xl font-bold text-amber-800 dark:text-amber-100">Vocabulary</h2>
        <button
          onClick={onBack}
          className="text-sm font-mono text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 border border-gray-200 dark:border-gray-800"
        >
          ← Back
        </button>
      </div>

      {error && <p className="text-red-600 dark:text-red-400 font-mono text-sm text-center py-8">{error}</p>}

      <div className="flex-1 overflow-y-auto">
        <table className="w-full text-sm font-mono">
          <thead className="sticky top-0 bg-white dark:bg-gray-950">
            <tr className="border-b border-gray-200 dark:border-gray-800">
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Original</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Corrected</th>
              <th className="text-left py-3 px-4 text-gray-500 uppercase tracking-wider text-xs font-medium">Category</th>
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
