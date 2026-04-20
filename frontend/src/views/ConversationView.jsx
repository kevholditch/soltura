import { useState, useRef, useEffect } from 'react'
import { MessageSquare, BookOpen } from 'lucide-react'
import Message from '../components/Message.jsx'

export default function ConversationView({ sessionId, topic, history, onHistoryUpdate, onEnd }) {
  const [messages, setMessages] = useState(() =>
    history.map((h, i) => ({
      id: String(i),
      role: h.role,
      text: h.content,
      corrections: [],
      isLoading: false,
      isStreaming: false,
      isError: false,
    }))
  )
  const [inputText, setInputText] = useState('')
  const [submitDisabled, setSubmitDisabled] = useState(false)
  const [ending, setEnding] = useState(false)
  const [activeTab, setActiveTab] = useState('chat')
  const [unreadCount, setUnreadCount] = useState(0)
  const threadRef = useRef(null)
  const inputRef = useRef(null)
  const activeTabRef = useRef('chat')

  useEffect(() => {
    activeTabRef.current = activeTab
  }, [activeTab])

  useEffect(() => {
    if (threadRef.current) {
      threadRef.current.scrollTop = threadRef.current.scrollHeight
    }
  }, [messages])

  async function submitTurn() {
    const text = inputText.trim()
    if (!text || !sessionId || submitDisabled) return

    const userMsgId = `${Date.now()}-user`
    const agentMsgId = `${Date.now()}-agent`

    setMessages(prev => [
      ...prev,
      { id: userMsgId, role: 'user', text, corrections: [], isLoading: false, isStreaming: false, isError: false },
      { id: agentMsgId, role: 'assistant', text: '', corrections: [], isLoading: true, isStreaming: false, isError: false },
    ])
    setInputText('')
    setSubmitDisabled(true)

    const updatedHistory = [...history, { role: 'user', content: text }]

    try {
      const response = await fetch(`/api/sessions/${sessionId}/turns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_text: text, history: updatedHistory.slice(-40) }),
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let fullText = ''
      let firstChunk = true

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
              if (firstChunk) {
                firstChunk = false
                setMessages(prev => prev.map(m =>
                  m.id === agentMsgId ? { ...m, isLoading: false, isStreaming: true } : m
                ))
              }
              fullText += event.text
              setMessages(prev => prev.map(m =>
                m.id === agentMsgId ? { ...m, text: fullText } : m
              ))
            } else if (event.type === 'corrections') {
              setMessages(prev => prev.map(m =>
                m.id === agentMsgId ? { ...m, isStreaming: false, corrections: event.corrections } : m
              ))
              if (activeTabRef.current !== 'corrections' && event.corrections.length > 0) {
                setUnreadCount(prev => prev + event.corrections.length)
              }
            } else if (event.type === 'done') {
              onHistoryUpdate([...updatedHistory, { role: 'assistant', content: fullText }])
              setSubmitDisabled(false)
              inputRef.current?.focus()
            }
          } catch (_) { /* ignore malformed SSE lines */ }
        }
      }
    } catch (err) {
      console.error('submitTurn error:', err?.name, err?.message, err)
      setMessages(prev => prev.map(m =>
        m.id === agentMsgId
          ? { ...m, isLoading: false, isStreaming: false, isError: true, text: '[Error: could not get response. Please try again.]' }
          : m
      ))
      setSubmitDisabled(false)
      inputRef.current?.focus()
    }
  }

  async function endSession() {
    setEnding(true)
    try {
      await fetch(`/api/sessions/${sessionId}/end`, { method: 'POST' })
    } catch (_) { /* continue to summary even on error */ }
    onEnd()
  }

  const allCorrections = [...messages].reverse().flatMap(m => m.corrections ?? [])

  return (
    <div className="flex-1 flex flex-col max-w-4xl w-full mx-auto px-4 py-4" style={{ height: 'calc(100vh - 3.5rem)' }}>

      <div className="flex items-center justify-between mb-3 flex-shrink-0">
        <div className="flex items-center gap-3">
          <span className="text-xs font-mono text-gray-500 uppercase tracking-wider">Topic</span>
          <span className="font-fraunces text-amber-700 dark:text-amber-200 font-semibold text-lg">{topic}</span>
        </div>
        <button
          onClick={endSession}
          disabled={ending}
          className="text-sm font-mono text-gray-500 dark:text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-50 dark:hover:bg-gray-900 border border-gray-200 dark:border-gray-800 hover:border-red-200 dark:hover:border-red-900 disabled:opacity-50"
        >
          {ending ? 'Ending…' : 'End Session'}
        </button>
      </div>

      <div className="flex items-center gap-1 mb-3 flex-shrink-0 border-b border-gray-200 dark:border-gray-800 pb-2">
        <button
          onClick={() => setActiveTab('chat')}
          aria-label="Chat"
          className={`p-2 rounded-md transition-colors ${
            activeTab === 'chat'
              ? 'text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20'
              : 'text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
          }`}
        >
          <MessageSquare size={18} />
        </button>

        <button
          onClick={() => { setActiveTab('corrections'); setUnreadCount(0) }}
          aria-label="Grammar corrections"
          className={`relative p-2 rounded-md transition-colors ${
            activeTab === 'corrections'
              ? 'text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20'
              : 'text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
          }`}
        >
          <BookOpen size={18} />
          {unreadCount > 0 && (
            <span className="absolute -top-1 -right-1 min-w-[16px] h-4 px-1 rounded-full bg-[#C1440E] text-white text-[10px] font-mono font-bold flex items-center justify-center leading-none">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </button>
      </div>

      {activeTab === 'chat' && (
        <>
          <div
            ref={threadRef}
            className="flex-1 overflow-y-auto space-y-4 pr-1 pb-4 message-thread"
            style={{ minHeight: 0 }}
          >
            {messages.map(msg => (
              <Message key={msg.id} {...msg} />
            ))}
          </div>

          <div className="flex-shrink-0 mt-4 bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4">
            <textarea
              ref={inputRef}
              rows={3}
              value={inputText}
              onChange={e => setInputText(e.target.value)}
              onKeyDown={e => { if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') submitTurn() }}
              placeholder="Write in Spanish... (Cmd+Enter to send)"
              className="w-full bg-transparent text-gray-900 dark:text-gray-100 font-mono text-sm placeholder-gray-400 dark:placeholder-gray-600 focus:outline-none resize-none leading-relaxed"
            />
            <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-200 dark:border-gray-800">
              <span className="text-xs font-mono text-gray-400 dark:text-gray-600">Cmd+Enter to send</span>
              <button
                onClick={submitTurn}
                disabled={submitDisabled}
                className="font-mono text-sm px-5 py-2 rounded-lg text-white font-medium transition-all disabled:opacity-40 disabled:cursor-not-allowed"
                style={{ backgroundColor: '#C1440E' }}
                onMouseOver={e => { if (!submitDisabled) e.currentTarget.style.backgroundColor = '#a33a0c' }}
                onMouseOut={e => { e.currentTarget.style.backgroundColor = '#C1440E' }}
              >
                Send →
              </button>
            </div>
          </div>
        </>
      )}

      {activeTab === 'corrections' && (
        <div className="flex-1 overflow-y-auto pr-1 pb-4 message-thread" style={{ minHeight: 0 }}>
          {allCorrections.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <BookOpen size={32} className="text-gray-300 dark:text-gray-700 mb-3" />
              <p className="text-gray-400 dark:text-gray-600 font-mono text-sm">No corrections yet. Keep chatting!</p>
            </div>
          ) : (
            <div className="space-y-1 pt-1">
              {allCorrections.map((c, i) => (
                <div key={i} className="correction-item">
                  <div className="correction-row">
                    <span className="correction-original">{c.original}</span>
                    <span className="correction-arrow">→</span>
                    <span className="correction-corrected">{c.corrected}</span>
                    <span className={`correction-badge category-${c.category}`}>{c.category}</span>
                  </div>
                  <p className="correction-explanation">{c.explanation}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

    </div>
  )
}
