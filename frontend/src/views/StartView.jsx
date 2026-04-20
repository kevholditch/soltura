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
            onKeyDown={e => { if (e.key === 'Enter') startSession() }}
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
