import { useState, useEffect } from 'react'
import { Sun, Moon } from 'lucide-react'
import StartView from './views/StartView.jsx'
import ConversationView from './views/ConversationView.jsx'
import SummaryView from './views/SummaryView.jsx'
import VocabView from './views/VocabView.jsx'
import DrillView from './views/DrillView.jsx'

export default function App() {
  const [view, setView] = useState('start')
  const [sessionId, setSessionId] = useState(null)
  const [topic, setTopic] = useState('')
  const [history, setHistory] = useState([])
  const [theme, setTheme] = useState(() => localStorage.getItem('theme') ?? 'dark')

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    localStorage.setItem('theme', theme)
  }, [theme])

  function handleTopicSelected(selectedTopic) {
    setTopic(selectedTopic)
    setSessionId(null)
    setHistory([])
    setView('conversation')
  }

  function handleDrillsStart() {
    setView('drills')
  }

  return (
    <div className="bg-white dark:bg-gray-950 text-gray-900 dark:text-gray-100 min-h-screen flex flex-col">
      <nav className="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 sticky top-0 z-10">
        <div className="max-w-4xl mx-auto px-4 h-14 flex items-center justify-between">
          <span className="font-fraunces text-xl font-semibold text-amber-800 dark:text-amber-100 tracking-tight">Soltura</span>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setView('vocab')}
              className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 font-mono"
            >
              Vocabulary
            </button>
            <button
              onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}
              aria-label="Toggle theme"
              className="p-2 rounded-md text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
            </button>
          </div>
        </div>
      </nav>

      {view === 'start' && (
        <StartView onTopicSelected={handleTopicSelected} onDrillsStart={handleDrillsStart} />
      )}
      {view === 'conversation' && (
        <ConversationView
          sessionId={sessionId}
          topic={topic}
          history={history}
          onHistoryUpdate={setHistory}
          onSessionCreated={setSessionId}
          onEnd={() => setView('summary')}
        />
      )}
      {view === 'summary' && (
        <SummaryView
          sessionId={sessionId}
          onNewSession={() => {
            setSessionId(null)
            setTopic('')
            setHistory([])
            setView('start')
          }}
        />
      )}
      {view === 'vocab' && (
        <VocabView onBack={() => setView(sessionId ? 'conversation' : 'start')} />
      )}
      {view === 'drills' && (
        <DrillView onExit={() => setView('start')} />
      )}
    </div>
  )
}
