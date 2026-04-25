import { useEffect, useMemo, useState } from 'react'
import { ArrowLeft, MessageCirclePlus } from 'lucide-react'
import { marked } from 'marked'
import Message from '../components/Message.jsx'

function flattenCorrections(turns) {
  return turns.flatMap(turn => turn.corrections ?? [])
}

function transcriptMessages(turns, seedContent) {
  const messages = turns.flatMap(turn => [
    {
      id: `${turn.id}-user`,
      role: 'user',
      text: turn.user_text,
      corrections: [],
    },
    {
      id: `${turn.id}-assistant`,
      role: 'assistant',
      text: turn.agent_reply,
      corrections: turn.corrections ?? [],
    },
  ].filter(message => message.text))

  if (seedContent) {
    return [
      { id: 'session-seed', role: 'assistant', text: seedContent, corrections: [] },
      ...messages,
    ]
  }

  return messages
}

function markdownWithNewlines(text) {
  return (text || '').replaceAll('\\n', '\n')
}

export default function SessionReviewView({ sessionId, onBack, onStartNewChat }) {
  const [review, setReview] = useState(null)
  const [summary, setSummary] = useState(null)
  const [loading, setLoading] = useState(true)
  const [summaryLoading, setSummaryLoading] = useState(false)
  const [error, setError] = useState('')
  const [summaryError, setSummaryError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setSummaryLoading(true)
    setError('')
    setSummaryError('')
    setReview(null)
    setSummary(null)

    if (!sessionId) {
      setError('No session selected.')
      setLoading(false)
      return () => { cancelled = true }
    }

    fetch(`/api/sessions/${sessionId}/review`)
      .then(response => {
        if (!response.ok) throw new Error(`Review server error: ${response.status}`)
        return response.json()
      })
      .then(reviewData => {
        if (cancelled) return
        setReview(reviewData)
      })
      .catch(err => {
        if (cancelled) return
        setError(err.message)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    fetch(`/api/sessions/${sessionId}/summary`)
      .then(response => {
        if (!response.ok) throw new Error(`Summary server error: ${response.status}`)
        return response.json()
      })
      .then(summaryData => {
        if (cancelled) return
        setSummary(summaryData)
      })
      .catch(err => {
        if (cancelled) return
        setSummaryError(err.message)
      })
      .finally(() => {
        if (!cancelled) setSummaryLoading(false)
      })

    return () => { cancelled = true }
  }, [sessionId])

  const turns = review?.turns ?? []
  const messages = useMemo(() => transcriptMessages(turns, review?.session?.seed_content), [turns, review?.session?.seed_content])
  const corrections = useMemo(() => flattenCorrections(turns), [turns])
  const topic = review?.session?.topic ?? 'Session review'

  return (
    <main className="flex-1 w-full max-w-4xl mx-auto px-4 py-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-2 self-start font-mono text-sm text-gray-500 dark:text-gray-400 hover:text-amber-700 dark:hover:text-amber-300 transition-colors"
        >
          <ArrowLeft size={16} />
          Back to Journal
        </button>

        {review?.session?.topic && (
          <button
            type="button"
            onClick={() => onStartNewChat(review.session.topic)}
            className="inline-flex items-center justify-center gap-2 font-mono text-sm px-4 py-2 rounded-lg text-white font-medium transition-colors bg-orange-700 hover:bg-orange-800"
          >
            <MessageCirclePlus size={16} />
            Start new chat on this topic
          </button>
        )}
      </div>

      {loading && (
        <p className="text-gray-500 dark:text-gray-400 font-mono text-sm">Loading session review...</p>
      )}

      {error && (
        <div className="border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/30 rounded-lg p-4">
          <p className="text-red-700 dark:text-red-300 font-mono text-sm">Could not load session review: {error}</p>
        </div>
      )}

      {!loading && !error && review && (
        <>
          <header className="mb-8">
            <p className="font-mono text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-2">Session Review</p>
            <h2 className="font-fraunces text-4xl font-bold text-amber-800 dark:text-amber-100">{topic}</h2>
          </header>

          <section className="mb-10" aria-labelledby="review-transcript">
            <h3 id="review-transcript" className="font-fraunces text-2xl font-semibold text-amber-800 dark:text-amber-100 mb-4">Transcript</h3>
            {messages.length === 0 ? (
              <div className="border border-gray-200 dark:border-gray-800 rounded-lg p-6">
                <p className="text-gray-500 dark:text-gray-400 font-mono text-sm">No stored turns are available for this session.</p>
              </div>
            ) : (
              <div className="space-y-4 message-thread">
                {messages.map(message => (
                  <Message key={message.id} {...message} />
                ))}
              </div>
            )}
          </section>

          <section className="mb-10" aria-labelledby="review-corrections">
            <div className="flex items-baseline justify-between border-b border-gray-200 dark:border-gray-800 pb-2 mb-3">
              <h3 id="review-corrections" className="font-fraunces text-2xl font-semibold text-amber-800 dark:text-amber-100">Corrections</h3>
              <span className="font-mono text-xs text-gray-500 dark:text-gray-400">{corrections.length} total</span>
            </div>
            {corrections.length === 0 ? (
              <p className="text-gray-500 dark:text-gray-400 font-mono text-sm">No corrections were recorded for this session.</p>
            ) : (
              <div className="space-y-1">
                {corrections.map((correction, index) => (
                  <div key={correction.id ?? index} className="correction-item">
                    <div className="correction-row">
                      <span className="correction-original">{correction.original}</span>
                      <span className="correction-arrow">→</span>
                      <span className="correction-corrected">{correction.corrected}</span>
                      <span className={`correction-badge category-${correction.category}`}>{correction.category}</span>
                    </div>
                    <p className="correction-explanation">{correction.explanation}</p>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section aria-labelledby="review-summary">
            <h3 id="review-summary" className="font-fraunces text-2xl font-semibold text-amber-800 dark:text-amber-100 mb-4">Review</h3>
            {summaryError ? (
              <div className="border border-amber-200 dark:border-amber-900 bg-amber-50 dark:bg-amber-950/20 rounded-lg p-5">
                <p className="text-amber-800 dark:text-amber-200 font-mono text-sm">
                  Stored transcript and corrections are available, but the summary could not be loaded: {summaryError}
                </p>
              </div>
            ) : (
              <div
                className="summary-text text-gray-700 dark:text-gray-300 font-mono text-sm leading-relaxed border border-gray-200 dark:border-gray-800 rounded-lg p-5 bg-white dark:bg-gray-900"
                dangerouslySetInnerHTML={{ __html: marked.parse(markdownWithNewlines(summary?.summary || (summaryLoading ? 'Loading summary...' : 'No summary is available yet.'))) }}
              />
            )}
          </section>
        </>
      )}
    </main>
  )
}
