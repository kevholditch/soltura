import { useState, useRef, useEffect } from 'react'

export default function StartView({ onTopicSelected, onDrillsStart }) {
  const [topic, setTopic] = useState('')
  const [mode, setMode] = useState('chat')
  const inputRef = useRef(null)

  useEffect(() => {
    if (mode === 'chat') inputRef.current?.focus()
  }, [mode])

  function handleSubmit() {
    const trimmed = topic.trim()
    if (!trimmed) return
    onTopicSelected(trimmed)
  }

  return (
    <div className="flex-1 flex items-center justify-center px-4">
      <div className="w-full max-w-lg">
        <div className="text-center mb-12">
          <h1 className="font-fraunces text-6xl font-bold text-amber-800 dark:text-amber-100 mb-3 leading-tight">Soltura</h1>
          <p className="text-gray-500 dark:text-gray-400 text-lg font-mono">Your Spanish conversation partner</p>
        </div>

        {/* Mode toggle */}
        <div className="flex items-center justify-center mb-6">
          <div className="flex bg-gray-100 dark:bg-gray-800 rounded-lg p-1 gap-1">
            <button
              onClick={() => setMode('chat')}
              className={`px-5 py-2 rounded-md font-mono text-sm font-medium transition-colors ${
                mode === 'chat'
                  ? 'bg-white dark:bg-gray-950 text-amber-800 dark:text-amber-200 shadow-sm'
                  : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
              }`}
            >
              Chat
            </button>
            <button
              onClick={() => setMode('drills')}
              className={`px-5 py-2 rounded-md font-mono text-sm font-medium transition-colors ${
                mode === 'drills'
                  ? 'bg-white dark:bg-gray-950 text-amber-800 dark:text-amber-200 shadow-sm'
                  : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
              }`}
            >
              Drills
            </button>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-8 shadow-2xl">
          {mode === 'chat' ? (
            <>
              <label className="block text-sm font-mono text-gray-600 dark:text-gray-400 mb-3 uppercase tracking-wider">
                What would you like to talk about?
              </label>
              <input
                ref={inputRef}
                type="text"
                value={topic}
                onChange={e => setTopic(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') handleSubmit() }}
                placeholder="e.g. My weekend plans, favourite films, cooking..."
                className="w-full bg-gray-50 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg px-4 py-3 text-gray-900 dark:text-gray-100 font-mono text-sm placeholder-gray-400 dark:placeholder-gray-600 focus:outline-none focus:border-amber-600 focus:ring-1 focus:ring-amber-600 transition-colors mb-5"
              />
              <button
                onClick={handleSubmit}
                className="w-full text-white font-semibold py-3 px-6 rounded-lg transition-colors font-mono text-sm uppercase tracking-wider"
                style={{ backgroundColor: '#C1440E' }}
                onMouseOver={e => { e.currentTarget.style.backgroundColor = '#a33a0c' }}
                onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
              >
                Start Session
              </button>
            </>
          ) : (
            <>
              <p className="text-gray-600 dark:text-gray-400 font-mono text-sm mb-6 leading-relaxed">
                I'll analyse your most common mistakes and drill you on them until they stick. Each pattern is explained in Spanish, then you practise with questions until I'm confident you've got it.
              </p>
              <button
                onClick={onDrillsStart}
                className="w-full text-white font-semibold py-3 px-6 rounded-lg transition-colors font-mono text-sm uppercase tracking-wider"
                style={{ backgroundColor: '#C1440E' }}
                onMouseOver={e => { e.currentTarget.style.backgroundColor = '#a33a0c' }}
                onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
              >
                Start Drilling
              </button>
            </>
          )}
        </div>

        <p className="text-center text-gray-400 dark:text-gray-600 text-xs font-mono mt-6">
          {mode === 'chat' ? 'Press Enter to start · Cmd+Enter to send messages' : 'Requires at least one completed chat session'}
        </p>
      </div>
    </div>
  )
}
