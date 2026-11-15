import { MAX_RECORD_SECONDS, useAudioRecorder } from '../hooks/useAudioRecorder.js'

export default function AudioRecorder({ onRecorded, disabled }) {
  const { recording, seconds, level, error, start, stop } = useAudioRecorder(onRecorded)

  return (
    <div className="recorder">
      <button
        type="button"
        className={`icon-btn mic ${recording ? 'live' : ''}`}
        onClick={recording ? stop : start}
        disabled={disabled && !recording}
        aria-label={recording ? 'Stop recording' : 'Record from microphone'}
        title={recording ? 'Stop and identify' : 'Listen with microphone'}
      >
        {recording ? (
          <svg viewBox="0 0 24 24" width="20" height="20"><rect x="6" y="6" width="12" height="12" rx="2" fill="currentColor" /></svg>
        ) : (
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
            <rect x="9" y="3" width="6" height="12" rx="3" />
            <path d="M5 11a7 7 0 0 0 14 0M12 18v3" />
          </svg>
        )}
      </button>
      {recording && (
        <div className="rec-status" role="status">
          <span className="meter"><span style={{ width: `${Math.round(level * 100)}%` }} /></span>
          <span className="rec-time">{seconds}s / {MAX_RECORD_SECONDS}s</span>
        </div>
      )}
      {error && <div className="rec-error" role="alert">{error}</div>}
    </div>
  )
}
