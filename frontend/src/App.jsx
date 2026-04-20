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
    </div>
  )
}
