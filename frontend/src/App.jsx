import { useState, useEffect } from 'react'
import MenuBar from './components/MenuBar.jsx'
import StartView from './views/StartView.jsx'
import ConversationView from './views/ConversationView.jsx'
import SummaryView from './views/SummaryView.jsx'
import VocabView from './views/VocabView.jsx'
import DrillView from './views/DrillView.jsx'
import JournalView from './views/JournalView.jsx'
import SessionReviewView from './views/SessionReviewView.jsx'

export default function App() {
  const [view, setView] = useState('start')
  const [sessionId, setSessionId] = useState(null)
  const [reviewSessionId, setReviewSessionId] = useState(null)
  const [scopedDrillVocabIds, setScopedDrillVocabIds] = useState([])
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

  function handleNewSession() {
    setSessionId(null)
    setReviewSessionId(null)
    setTopic('')
    setHistory([])
    setView('start')
  }

  function handleDrillsStart() {
    setScopedDrillVocabIds([])
    setView('drills')
  }

  function handleJournalOpen() {
    setReviewSessionId(null)
    setView('journal')
  }

  function handleSessionReviewOpen(selectedSessionId) {
    setReviewSessionId(selectedSessionId)
    setView('sessionReview')
  }

  function handleScopedDrillStart(vocabIds) {
    setScopedDrillVocabIds(vocabIds ?? [])
    setView('drills')
  }

  return (
    <div className="bg-white dark:bg-gray-950 text-gray-900 dark:text-gray-100 min-h-screen flex flex-col">
      <MenuBar
        activeView={view}
        theme={theme}
        onNewSession={handleNewSession}
        onDrillsStart={handleDrillsStart}
        onVocabularyOpen={() => setView('vocab')}
        onJournalOpen={handleJournalOpen}
        onThemeToggle={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}
      />

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
        />
      )}
      {view === 'vocab' && (
        <VocabView
          onBack={() => setView(sessionId ? 'conversation' : 'start')}
          onDrillStart={handleScopedDrillStart}
        />
      )}
      {view === 'drills' && (
        <DrillView onExit={() => setView('start')} initialVocabIds={scopedDrillVocabIds} />
      )}
      {view === 'journal' && (
        <JournalView onSessionOpen={handleSessionReviewOpen} />
      )}
      {view === 'sessionReview' && (
        <SessionReviewView
          sessionId={reviewSessionId}
          onBack={handleJournalOpen}
          onStartNewChat={handleTopicSelected}
        />
      )}
    </div>
  )
}
