import { useState, useEffect, useRef } from 'react'
import LoadingBubble from '../components/LoadingBubble.jsx'

export default function DrillView({ onExit }) {
  const [phase, setPhase] = useState('loading')
  const [patternName, setPatternName] = useState('')
  const [explanation, setExplanation] = useState('')
  const [question, setQuestion] = useState('')
  const [vocabIds, setVocabIds] = useState([])
  const [answer, setAnswer] = useState('')
  const [feedbackText, setFeedbackText] = useState('')
  const [transitionText, setTransitionText] = useState('')
  const [showTransition, setShowTransition] = useState(false)
  const [drillHistory, setDrillHistory] = useState([])
  const [patternCount, setPatternCount] = useState(0)
  const [phrases, setPhrases] = useState([])
  const [phraseIndex, setPhraseIndex] = useState(0)
  const inputRef = useRef(null)

  // Fetch phrases and kick off first drill in parallel
  useEffect(() => {
    fetch('/api/drills/phrases')
      .then(r => r.json())
      .then(data => setPhrases(data))
      .catch(() => {})
    loadNextPattern()
  }, [])

  // Cycle phrases every 2.5s while loading
  useEffect(() => {
    if (phase !== 'loading' || phrases.length === 0) return
    const id = setInterval(() => {
      setPhraseIndex(i => (i + 1) % phrases.length)
    }, 2500)
    return () => clearInterval(id)
  }, [phase, phrases])

  useEffect(() => {
    if (phase === 'question') inputRef.current?.focus()
  }, [phase])

  async function loadNextPattern() {
    setPhase('loading')
    setPhraseIndex(0)
    setTransitionText('')
    setShowTransition(false)
    try {
      const res = await fetch('/api/drills/start', { method: 'POST' })
      if (!res.ok) throw new Error(`Server error: ${res.status}`)
      const data = await res.json()
      if (data.all_done) {
        setPhase('all_done')
        return
      }
      setPatternName(data.pattern_name)
      setExplanation(data.explanation)
      setQuestion(data.question)
      setVocabIds(data.vocab_ids ?? [])
      setDrillHistory([
        { role: 'assistant', content: data.explanation },
        { role: 'assistant', content: data.question },
      ])
      setAnswer('')
      setFeedbackText('')
      setPatternCount(c => c + 1)
      setPhase('question')
    } catch (err) {
      console.error('loadNextPattern error:', err)
      setPhase('all_done')
    }
  }

  async function submitAnswer() {
    const trimmed = answer.trim()
    if (!trimmed) return
    setFeedbackText('')
    setTransitionText('')
    setShowTransition(false)
    setPhase('feedback')

    const updatedHistory = [...drillHistory, { role: 'user', content: trimmed }]

    try {
      const response = await fetch('/api/drills/turn', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          answer: trimmed,
          history: updatedHistory,
          pattern_name: patternName,
          explanation,
          question,
          vocab_ids: vocabIds,
        }),
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let fullFeedback = ''
      let fullTransition = ''
      let drillResult = null

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop()

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const jsonStr = line.slice(6)
          if (jsonStr === '[DONE]') break
          try {
            const event = JSON.parse(jsonStr)
            if (event.type === 'chunk') {
              fullFeedback += event.text
              setFeedbackText(fullFeedback)
            } else if (event.type === 'transition_start') {
              setShowTransition(true)
            } else if (event.type === 'transition_chunk') {
              fullTransition += event.text
              setTransitionText(fullTransition)
            } else if (event.type === 'drill_result') {
              drillResult = event
            }
          } catch (_) { /* ignore malformed lines */ }
        }
      }

      if (drillResult) {
        const newHistory = [
          ...updatedHistory,
          { role: 'assistant', content: fullFeedback },
        ]

        if (drillResult.mastered) {
          setDrillHistory(newHistory)
          setTimeout(() => loadNextPattern(), 2500)
        } else {
          const nextQ = drillResult.next_question
          if (nextQ) {
            setQuestion(nextQ)
            setDrillHistory([...newHistory, { role: 'assistant', content: nextQ }])
          }
          setAnswer('')
          setPhase('question')
        }
      }
    } catch (err) {
      console.error('submitAnswer error:', err)
      setPhase('question')
    }
  }

  const currentPhrase = phrases.length > 0 ? phrases[phraseIndex] : ''

  return (
    <div className="flex-1 flex flex-col max-w-2xl w-full mx-auto px-4 py-6" style={{ height: 'calc(100vh - 3.5rem)' }}>

      {/* Header — always visible except all_done */}
      {phase !== 'all_done' && (
        <div className="flex items-center justify-between mb-4 flex-shrink-0">
          <button
            onClick={onExit}
            className="text-sm font-mono text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
          >
            ← Exit drills
          </button>
          {phase !== 'loading' && patternName && (
            <span className="text-xs font-mono text-gray-400 dark:text-gray-600 uppercase tracking-wider">
              Pattern {patternCount} · {patternName}
            </span>
          )}
        </div>
      )}

      {/* Loading */}
      {phase === 'loading' && (
        <div className="flex-1 flex flex-col" style={{ minHeight: 0 }}>
          <div className="agent-bubble font-mono text-sm text-gray-500 dark:text-gray-400 min-h-[3rem] flex items-center">
            {currentPhrase || <LoadingBubble />}
          </div>
        </div>
      )}

      {/* Question / Feedback */}
      {(phase === 'question' || phase === 'feedback') && (
        <div className="flex-1 flex flex-col overflow-y-auto" style={{ minHeight: 0 }}>
          <div className="space-y-4 pb-4">
            <div
              className="agent-bubble"
              dangerouslySetInnerHTML={{ __html: explanation }}
            />
            <div className="agent-bubble border-amber-200 dark:border-amber-800">
              {question}
            </div>

            {phase === 'feedback' && (
              <>
                <div className="agent-bubble">
                  {feedbackText ? feedbackText : <LoadingBubble />}
                </div>

                {showTransition && (
                  <>
                    <div className="flex items-center gap-3 py-1">
                      <div className="flex-1 border-t border-gray-200 dark:border-gray-700" />
                      <span className="text-xs font-mono text-green-600 dark:text-green-400 uppercase tracking-wider">✓ Dominado</span>
                      <div className="flex-1 border-t border-gray-200 dark:border-gray-700" />
                    </div>
                    <div className="agent-bubble border-green-200 dark:border-green-900">
                      {transitionText ? transitionText : <LoadingBubble />}
                    </div>
                  </>
                )}
              </>
            )}
          </div>

          {phase === 'question' && (
            <div className="flex-shrink-0 mt-auto bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4">
              <textarea
                ref={inputRef}
                rows={2}
                value={answer}
                onChange={e => setAnswer(e.target.value)}
                onKeyDown={e => { if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') submitAnswer() }}
                placeholder="Type your answer in Spanish… (Cmd+Enter to send)"
                className="w-full bg-transparent text-gray-900 dark:text-gray-100 font-mono text-sm placeholder-gray-400 dark:placeholder-gray-600 focus:outline-none resize-none leading-relaxed"
              />
              <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-200 dark:border-gray-800">
                <span className="text-xs font-mono text-gray-400 dark:text-gray-600">Cmd+Enter to send</span>
                <button
                  onClick={submitAnswer}
                  className="font-mono text-sm px-5 py-2 rounded-lg text-white font-medium transition-all"
                  style={{ backgroundColor: '#C1440E' }}
                  onMouseOver={e => { e.currentTarget.style.backgroundColor = '#a33a0c' }}
                  onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
                >
                  Answer →
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* All done */}
      {phase === 'all_done' && (
        <div className="flex-1 flex flex-col items-center justify-center text-center px-4">
          <p className="font-fraunces text-3xl text-amber-700 dark:text-amber-300 mb-4">All done!</p>
          <p className="font-mono text-sm text-gray-500 dark:text-gray-400 mb-8 max-w-sm">
            You've mastered all your current error patterns. Keep chatting to discover new ones.
          </p>
          <button
            onClick={onExit}
            className="font-mono text-sm px-6 py-3 rounded-lg text-white font-medium"
            style={{ backgroundColor: '#C1440E' }}
            onMouseOver={e => { e.currentTarget.style.backgroundColor = '#a33a0c' }}
            onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
          >
            Back to Home
          </button>
        </div>
      )}

    </div>
  )
}
