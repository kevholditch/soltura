import { useState, useRef, useEffect } from 'react'
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
  const threadRef = useRef(null)
  const inputRef = useRef(null)

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

  return (
    <div className="flex-1 flex flex-col max-w-4xl w-full mx-auto px-4 py-4" style={{ height: 'calc(100vh - 3.5rem)' }}>
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <div className="flex items-center gap-3">
          <span className="text-xs font-mono text-gray-500 uppercase tracking-wider">Topic</span>
          <span className="font-fraunces text-amber-200 font-semibold text-lg">{topic}</span>
        </div>
        <button
          onClick={endSession}
          disabled={ending}
          className="text-sm font-mono text-gray-400 hover:text-red-400 transition-colors px-3 py-1.5 rounded-md hover:bg-gray-900 border border-gray-800 hover:border-red-900 disabled:opacity-50"
        >
          {ending ? 'Ending…' : 'End Session'}
        </button>
      </div>

      <div
        ref={threadRef}
        className="flex-1 overflow-y-auto space-y-4 pr-1 pb-4 message-thread"
        style={{ minHeight: 0 }}
      >
        {messages.map(msg => (
          <Message key={msg.id} {...msg} />
        ))}
      </div>

      <div className="flex-shrink-0 mt-4 bg-gray-900 border border-gray-800 rounded-xl p-4">
        <textarea
          ref={inputRef}
          rows={3}
          value={inputText}
          onChange={e => setInputText(e.target.value)}
          onKeyDown={e => { if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') submitTurn() }}
          placeholder="Write in Spanish... (Cmd+Enter to send)"
          className="w-full bg-transparent text-gray-100 font-mono text-sm placeholder-gray-600 focus:outline-none resize-none leading-relaxed"
        />
        <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-800">
          <span className="text-xs font-mono text-gray-600">Cmd+Enter to send</span>
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
    </div>
  )
}
