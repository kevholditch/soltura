import { useState, useEffect, useRef } from 'react'
import LoadingBubble from '../components/LoadingBubble.jsx'

const BLANK_PATTERN = /_{3,}/g

function toLLMHistory(history) {
  return history
    .filter(item => item.role === 'assistant' || item.role === 'user')
    .map(item => ({ role: item.role, content: item.content }))
}

function parseQuestionTemplate(text) {
  const questionText = text || ''
  const segments = []
  let lastIndex = 0
  let blankCount = 0

  BLANK_PATTERN.lastIndex = 0
  let match = BLANK_PATTERN.exec(questionText)

  for (; match !== null; match = BLANK_PATTERN.exec(questionText)) {
    if (match.index > lastIndex) {
      segments.push({ type: 'text', value: questionText.slice(lastIndex, match.index) })
    }
    segments.push({ type: 'blank' })
    blankCount += 1
    lastIndex = match.index + match[0].length
  }

  if (lastIndex < questionText.length) {
    segments.push({ type: 'text', value: questionText.slice(lastIndex) })
  }

  if (blankCount === 0) {
    return {
      segments: [{ type: 'text', value: questionText }],
      blankCount: 1,
      hasInlineBlanks: false,
    }
  }

  return {
    segments,
    blankCount,
    hasInlineBlanks: true,
  }
}

function composeAnswer(blankAnswers) {
  return blankAnswers
    .map(value => value.trim())
    .filter(Boolean)
    .join(' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function SubmittedQuestionContent({ item }) {
  const template = parseQuestionTemplate(item.content)
  const submittedAnswers = item.submittedAnswers ?? []

  if (!item.submitted) {
    return <div>{item.content}</div>
  }

  if (template.hasInlineBlanks) {
    let blankIndex = -1
    return (
      <div className="drill-inline-question">
        {template.segments.map((segment, segmentIdx) => {
          if (segment.type === 'text') {
            return <span key={segmentIdx}>{segment.value}</span>
          }

          blankIndex += 1
          return (
            <input
              key={segmentIdx}
              type="text"
              value={submittedAnswers[blankIndex] ?? ''}
              disabled
              readOnly
              className="drill-inline-input drill-inline-input-submitted"
              aria-label={`Submitted answer ${blankIndex + 1}`}
            />
          )
        })}
      </div>
    )
  }

  return (
    <>
      <div>{item.content}</div>
      <input
        type="text"
        value={item.answer ?? composeAnswer(submittedAnswers)}
        disabled
        readOnly
        className="drill-inline-input drill-inline-input-full drill-inline-input-submitted"
        aria-label="Submitted answer"
      />
    </>
  )
}

function markLastQuestion(history, attrs) {
  for (let i = history.length - 1; i >= 0; i -= 1) {
    if (history[i].kind === 'question') {
      const copy = [...history]
      copy[i] = { ...copy[i], ...attrs }
      return copy
    }
  }
  return history
}

export default function DrillView({ onExit, initialVocabIds = [] }) {
  const [phase, setPhase] = useState('loading')
  const [patternName, setPatternName] = useState('')
  const [explanation, setExplanation] = useState('')
  const [question, setQuestion] = useState('')
  const [vocabIds, setVocabIds] = useState([])
  const [blankAnswers, setBlankAnswers] = useState([''])
  const [feedbackText, setFeedbackText] = useState('')
  const [transitionText, setTransitionText] = useState('')
  const [drillHistory, setDrillHistory] = useState([])
  const [patternCount, setPatternCount] = useState(0)
  const [phrases, setPhrases] = useState([])
  const [phraseIndex, setPhraseIndex] = useState(0)
  const blankRefs = useRef([])
  const historyRef = useRef(null)

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
    const template = parseQuestionTemplate(question)
    setBlankAnswers(Array.from({ length: template.blankCount }, () => ''))
    blankRefs.current = []
  }, [question])

  useEffect(() => {
    if (phase !== 'question') return
    setTimeout(() => {
      blankRefs.current[0]?.focus()
    }, 0)
  }, [phase, question])

  useEffect(() => {
    if (!historyRef.current) return
    historyRef.current.scrollTop = historyRef.current.scrollHeight
  }, [drillHistory, feedbackText, transitionText, phase])

  async function loadNextPattern() {
    setPhase('loading')
    setPhraseIndex(0)
    setTransitionText('')
    try {
      const body = initialVocabIds.length > 0 ? JSON.stringify({ vocab_ids: initialVocabIds }) : undefined
      const res = await fetch('/api/drills/start', {
        method: 'POST',
        headers: body ? { 'Content-Type': 'application/json' } : undefined,
        body,
      })
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
        { role: 'assistant', kind: 'explanation', content: data.explanation },
        { role: 'assistant', kind: 'question', content: data.question },
      ])
      setFeedbackText('')
      setPatternCount(c => c + 1)
      setPhase('question')
    } catch (err) {
      console.error('loadNextPattern error:', err)
      setPhase('all_done')
    }
  }

  function handleBlankChange(blankIndex, value) {
    setBlankAnswers(current => {
      const next = [...current]
      next[blankIndex] = value
      return next
    })
  }

  function focusBlank(blankIndex) {
    const target = blankRefs.current[blankIndex]
    if (target) target.focus()
  }

  function handleBlankKeyDown(event, blankIndex) {
    if (event.key === 'Enter') {
      event.preventDefault()
      submitAnswer()
      return
    }

    if (event.key === 'Tab') {
      event.preventDefault()
      if (blankAnswers.length <= 1) return
      const delta = event.shiftKey ? -1 : 1
      const nextIndex = (blankIndex + delta + blankAnswers.length) % blankAnswers.length
      focusBlank(nextIndex)
    }
  }

  async function submitAnswer() {
    if (phase !== 'question') return

    const allBlanksFilled = blankAnswers.every(value => value.trim().length > 0)
    const trimmed = composeAnswer(blankAnswers)
    if (!allBlanksFilled || !trimmed) return

    setFeedbackText('')
    setTransitionText('')

    const submittedAnswers = blankAnswers.map(value => value.trim())
    const submittedHistory = markLastQuestion(drillHistory, {
      submitted: true,
      correct: null,
      submittedAnswers,
      answer: trimmed,
    })
    setDrillHistory(submittedHistory)
    setPhase('feedback')

    const llmHistory = [...submittedHistory, { role: 'user', kind: 'answer', content: trimmed }]

    try {
      const response = await fetch('/api/drills/turn', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          answer: trimmed,
          history: toLLMHistory(llmHistory),
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
      let markReceived = false
      let markedCorrect = null
      let liveHistory = submittedHistory

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
            if (event.type === 'mark') {
              markReceived = true
              markedCorrect = !!event.correct
              liveHistory = markLastQuestion(liveHistory, { submitted: true, correct: markedCorrect })
              setDrillHistory(liveHistory)
            } else if (event.type === 'chunk') {
              fullFeedback += event.text
              setFeedbackText(fullFeedback)
            } else if (event.type === 'transition_chunk') {
              fullTransition += event.text
              setTransitionText(fullTransition)
            } else if (event.type === 'drill_result') {
              drillResult = event
            }
          } catch (_) {
            // ignore malformed lines
          }
        }
      }

      if (drillResult) {
        const effectiveCorrect = markReceived ? markedCorrect : !!drillResult.correct
        if (!markReceived) {
          liveHistory = markLastQuestion(liveHistory, { submitted: true, correct: effectiveCorrect })
          setDrillHistory(liveHistory)
        }

        const feedbackMessage = {
          role: 'assistant',
          kind: 'feedback',
          correct: effectiveCorrect,
          content: fullFeedback || (effectiveCorrect ? 'Muy bien.' : 'Casi. Vamos a intentarlo de nuevo.'),
        }

        const nextHistory = [...liveHistory, feedbackMessage]

        if (drillResult.mastered) {
          const masteredHistory = [
            ...nextHistory,
            { role: 'assistant', kind: 'status', correct: true, content: '✓ Dominado' },
            ...(fullTransition ? [{ role: 'assistant', kind: 'transition', content: fullTransition }] : []),
          ]
          setDrillHistory(masteredHistory)
          setTimeout(() => loadNextPattern(), 2500)
        } else {
          const nextQ = drillResult.next_question
          const continuedHistory = [
            ...nextHistory,
            ...(nextQ ? [{ role: 'assistant', kind: 'question', content: nextQ }] : []),
          ]

          setDrillHistory(continuedHistory)
          if (nextQ) {
            setQuestion(nextQ)
            const nextTemplate = parseQuestionTemplate(nextQ)
            setBlankAnswers(Array.from({ length: nextTemplate.blankCount }, () => ''))
            blankRefs.current = []
          }
          setPhase('question')
        }
      }
    } catch (err) {
      console.error('submitAnswer error:', err)
      setDrillHistory(current => markLastQuestion(current, { submitted: false, correct: null }))
      setPhase('question')
    }
  }

  const currentPhrase = phrases.length > 0 ? phrases[phraseIndex] : ''
  let activeQuestionIndex = -1
  for (let i = drillHistory.length - 1; i >= 0; i -= 1) {
    if (drillHistory[i].kind === 'question') {
      activeQuestionIndex = i
      break
    }
  }

  const template = parseQuestionTemplate(question)
  const allBlanksFilled = blankAnswers.length > 0 && blankAnswers.every(value => value.trim().length > 0)

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

      {/* Drill transcript */}
      {(phase === 'question' || phase === 'feedback') && (
        <div className="flex-1 flex flex-col" style={{ minHeight: 0 }}>
          <div ref={historyRef} className="flex-1 overflow-y-auto message-thread pb-4">
            <div className="space-y-4">
              {drillHistory.map((item, idx) => {
                if (item.role === 'user') {
                  return null
                }

                if (item.kind === 'status') {
                  const statusClass = item.correct
                    ? 'drill-status-pill drill-status-pill-correct'
                    : 'drill-status-pill drill-status-pill-wrong'

                  return (
                    <div key={idx} className="flex items-center gap-3 py-1">
                      <div className="flex-1 border-t border-gray-200 dark:border-gray-700" />
                      <span className={statusClass}>
                        {item.content}
                      </span>
                      <div className="flex-1 border-t border-gray-200 dark:border-gray-700" />
                    </div>
                  )
                }

                let bubbleClass = 'agent-bubble'
                if (item.kind === 'question') bubbleClass += ' border-amber-200 dark:border-amber-800'
                if (item.kind === 'transition') bubbleClass += ' border-green-200 dark:border-green-900'
                if (item.kind === 'feedback' && item.correct) bubbleClass += ' border-green-200 dark:border-green-900'
                if (item.kind === 'feedback' && item.correct === false) bubbleClass += ' border-red-200 dark:border-red-900'
                if (item.kind === 'question' && item.submitted) bubbleClass += ' drill-question-bubble-submitted'
                if (item.kind === 'question' && item.correct === true) bubbleClass += ' drill-question-bubble-correct'
                if (item.kind === 'question' && item.correct === false) bubbleClass += ' drill-question-bubble-wrong'

                const isActiveQuestion = item.kind === 'question' && idx === activeQuestionIndex
                if (isActiveQuestion) {
                  const inputDisabled = phase !== 'question' || !!item.submitted
                  let blankIndex = -1
                  const showQuestionMark = item.correct === true || item.correct === false

                  return (
                    <div key={idx} className="agent-message">
                      <div className="drill-question-row">
                        <div className={`${bubbleClass} ${!item.submitted && phase === 'question' ? 'drill-question-bubble-active' : ''}`}>
                          {template.hasInlineBlanks ? (
                            <div className="drill-inline-question">
                              {template.segments.map((segment, segmentIdx) => {
                                if (segment.type === 'text') {
                                  return <span key={segmentIdx}>{segment.value}</span>
                                }

                                blankIndex += 1
                                const currentBlank = blankIndex
                                const value = blankAnswers[currentBlank] ?? ''
                                return (
                                  <input
                                    key={segmentIdx}
                                    ref={el => { blankRefs.current[currentBlank] = el }}
                                    type="text"
                                    value={value}
                                    disabled={inputDisabled}
                                    onChange={e => handleBlankChange(currentBlank, e.target.value)}
                                    onKeyDown={e => handleBlankKeyDown(e, currentBlank)}
                                    className={`drill-inline-input ${inputDisabled ? 'drill-inline-input-submitted' : ''}`}
                                    aria-label={`Blank ${currentBlank + 1}`}
                                  />
                                )
                              })}
                            </div>
                          ) : (
                            <>
                              <div>{item.content}</div>
                              <input
                                ref={el => { blankRefs.current[0] = el }}
                                type="text"
                                value={blankAnswers[0] ?? ''}
                                disabled={inputDisabled}
                                onChange={e => handleBlankChange(0, e.target.value)}
                                onKeyDown={e => handleBlankKeyDown(e, 0)}
                                className={`drill-inline-input drill-inline-input-full ${inputDisabled ? 'drill-inline-input-submitted' : ''}`}
                                aria-label="Answer"
                              />
                            </>
                          )}

                          {phase === 'question' && !item.submitted && (
                            <div className="drill-question-controls">
                              <span className="drill-question-hint">Enter to submit · Tab / Shift+Tab to move</span>
                              <button
                                onClick={submitAnswer}
                                disabled={!allBlanksFilled}
                                className="drill-submit-btn"
                              >
                                Submit
                              </button>
                            </div>
                          )}
                        </div>

                        {showQuestionMark && (
                          <span className={`drill-question-mark ${item.correct ? 'drill-question-mark-correct' : 'drill-question-mark-wrong'}`}>
                            {item.correct ? '✓' : '✗'}
                          </span>
                        )}
                      </div>
                    </div>
                  )
                }

                if (item.kind === 'question') {
                  const showQuestionMark = item.correct === true || item.correct === false
                  return (
                    <div key={idx} className="agent-message">
                      <div className="drill-question-row">
                        <div className={bubbleClass}>
                          <SubmittedQuestionContent item={item} />
                        </div>
                        {showQuestionMark && (
                          <span className={`drill-question-mark ${item.correct ? 'drill-question-mark-correct' : 'drill-question-mark-wrong'}`}>
                            {item.correct ? '✓' : '✗'}
                          </span>
                        )}
                      </div>
                    </div>
                  )
                }

                return (
                  <div key={idx} className="agent-message">
                    <div className={bubbleClass}>{item.content}</div>
                  </div>
                )
              })}

              {phase === 'feedback' && (
                <div className="agent-message">
                  <div className="agent-bubble streaming">
                    {feedbackText ? feedbackText : <LoadingBubble />}
                  </div>
                </div>
              )}

              {phase === 'feedback' && transitionText && (
                <div className="agent-message">
                  <div className="agent-bubble border-green-200 dark:border-green-900 streaming">
                    {transitionText}
                  </div>
                </div>
              )}
            </div>
          </div>
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
