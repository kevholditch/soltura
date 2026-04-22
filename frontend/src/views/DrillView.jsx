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
  const [drillHistory, setDrillHistory] = useState([])
  const [patternCount, setPatternCount] = useState(0)
  const inputRef = useRef(null)

  useEffect(() => {
    loadNextPattern()
  }, [])

  useEffect(() => {
    if (phase === 'question') inputRef.current?.focus()
  }, [phase])

  async function loadNextPattern() {
    setPhase('loading')
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
          setPhase('mastered')
          setTimeout(() => loadNextPattern(), 2000)
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

  return (
    <div className="flex-1 flex flex-col max-w-2xl w-full mx-auto px-4 py-6" style={{ height: 'calc(100vh - 3.5rem)' }}>

      {phase !== 'loading' && (
        <div className="flex items-center justify-between mb-4 flex-shrink-0">
          <button
            onClick={onExit}
            className="text-sm font-mono text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
          >
            ← Exit drills
          </button>
          {patternName && (
            <span className="text-xs font-mono text-gray-400 dark:text-gray-600 uppercase tracking-wider">
              Pattern {patternCount} · {patternName}
            </span>
          )}
        </div>
      )}

      {phase === 'loading' && (
        <div className="flex-1 flex flex-col items-center justify-center">
          <div className="agent-bubble w-full max-w-lg">
            <LoadingBubble />
          </div>
          <p className="text-xs font-mono text-gray-500 dark:text-gray-500 mt-3">Analysing your mistakes…</p>
        </div>
      )}

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
              <div className="agent-bubble">
                {feedbackText ? feedbackText : <LoadingBubble />}
              </div>
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

      {phase === 'mastered' && (
        <div className="flex-1 flex flex-col items-center justify-center text-center px-4">
          <div className="text-4xl mb-4 text-green-500">✓</div>
          <p className="font-fraunces text-2xl text-amber-700 dark:text-amber-300 mb-2">Pattern mastered!</p>
          <p className="font-mono text-sm text-gray-500 dark:text-gray-400">{patternName}</p>
          <p className="font-mono text-xs text-gray-400 dark:text-gray-600 mt-4">Loading next pattern…</p>
        </div>
      )}

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
