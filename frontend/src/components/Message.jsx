import { marked } from 'marked'
import LoadingBubble from './LoadingBubble.jsx'

export default function Message({ role, text, corrections = [], isLoading = false, isStreaming = false, isError = false }) {
  if (role === 'user') {
    return (
      <div className="user-message">
        <div className="user-bubble">{text}</div>
      </div>
    )
  }

  return (
    <div className="agent-message">
      {isLoading ? (
        <LoadingBubble />
      ) : (
        <div
          className={`agent-bubble${isStreaming ? ' streaming' : ''}${isError ? ' error-bubble' : ''}`}
          dangerouslySetInnerHTML={{ __html: marked.parse(text || '') }}
        />
      )}
    </div>
  )
}
