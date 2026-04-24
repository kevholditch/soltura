import { useState, useEffect } from 'react'
import { marked } from 'marked'

export default function SummaryView({ sessionId }) {
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
          <h2 className="font-fraunces text-4xl font-bold text-amber-800 dark:text-amber-100 mb-2">Session Complete</h2>
          <p className="text-gray-500 font-mono text-sm">Here's how you did</p>
        </div>

        {error && <p className="text-red-600 dark:text-red-400 font-mono text-sm text-center mb-4">{error}</p>}

        <div className="grid grid-cols-2 gap-4 mb-8">
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-6 text-center">
            <div className="font-fraunces text-5xl font-bold text-amber-600 dark:text-amber-300 mb-2">{data?.turn_count ?? '—'}</div>
            <div className="text-gray-500 font-mono text-xs uppercase tracking-wider">Turns</div>
          </div>
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-6 text-center">
            <div className="font-fraunces text-5xl font-bold text-orange-600 dark:text-orange-400 mb-2">{data?.correction_count ?? '—'}</div>
            <div className="text-gray-500 font-mono text-xs uppercase tracking-wider">Corrections</div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-6 mb-8">
          <h3 className="font-fraunces text-lg text-amber-700 dark:text-amber-200 mb-4 font-semibold">Feedback</h3>
          <div
            className="summary-text text-gray-700 dark:text-gray-300 font-mono text-sm leading-relaxed"
            dangerouslySetInnerHTML={{ __html: marked.parse(data?.summary || 'Loading…') }}
          />
        </div>

      </div>
    </div>
  )
}
