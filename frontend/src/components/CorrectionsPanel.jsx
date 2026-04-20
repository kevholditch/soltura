export default function CorrectionsPanel({ corrections }) {
  if (!corrections || corrections.length === 0) return null

  return (
    <div className="corrections-panel">
      <div className="corrections-heading">
        {corrections.length === 1 ? '1 correction' : `${corrections.length} corrections`}
      </div>
      {corrections.map((c, i) => (
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
  )
}
