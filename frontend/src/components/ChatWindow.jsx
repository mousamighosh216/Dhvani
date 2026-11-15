import { useCallback, useEffect, useRef, useState } from 'react'
import { recognizeAudio } from '../services/apiClient.js'
import AudioRecorder from './AudioRecorder.jsx'
import ResultCard from './ResultCard.jsx'

let nextId = 1
const WELCOME = {
  id: 0,
  role: 'assistant',
  kind: 'text',
  text: 'Hi, I’m Dhwani. Tap the microphone to listen, or drop in an audio clip, and I’ll tell you what song it is.',
}

export default function ChatWindow() {
  const [messages, setMessages] = useState([WELCOME])
  const [busy, setBusy] = useState(false)
  const [dragging, setDragging] = useState(false)
  const endRef = useRef(null)
  const fileRef = useRef(null)

  useEffect(() => {
    endRef.current?.scrollIntoView?.({ behavior: 'smooth' })
  }, [messages])

  const push = (m) => setMessages((prev) => [...prev, { id: nextId++, ...m }])
  const replace = (id, m) => setMessages((prev) => prev.map((x) => (x.id === id ? { id, ...m } : x)))

  const identify = useCallback(async (blob, filename) => {
    setBusy(true)
    push({ role: 'user', kind: 'audio', url: URL.createObjectURL(blob), label: filename })
    const loadingId = nextId++
    setMessages((prev) => [...prev, { id: loadingId, role: 'assistant', kind: 'loading' }])
    try {
      const data = await recognizeAudio(blob, filename)
      if (data.matched) {
        replace(loadingId, { role: 'assistant', kind: 'result', result: data })
      } else {
        replace(loadingId, {
          role: 'assistant',
          kind: 'text',
          text: 'I couldn’t find a match. Try a longer clip (5–10 seconds), get closer to the sound, or check that the song has been added to the catalog.',
        })
      }
    } catch (e) {
      replace(loadingId, { role: 'assistant', kind: 'error', text: e.message })
    } finally {
      setBusy(false)
    }
  }, [])

  const onFiles = (files) => {
    const file = files?.[0]
    if (file && !busy) identify(file, file.name)
  }

  return (
    <section
      className={`chat ${dragging ? 'dragging' : ''}`}
      onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => { e.preventDefault(); setDragging(false); onFiles(e.dataTransfer.files) }}
    >
      <div className="messages" aria-live="polite">
        {messages.map((m) => (
          <div key={m.id} className={`msg ${m.role}`}>
            {m.kind === 'text' && <p className="bubble">{m.text}</p>}
            {m.kind === 'error' && <p className="bubble error">{m.text}</p>}
            {m.kind === 'audio' && (
              <div className="bubble audio">
                <span className="fname">{m.label}</span>
                <audio controls src={m.url} />
              </div>
            )}
            {m.kind === 'loading' && (
              <p className="bubble loading" role="status">
                Listening<span className="dots"><i /><i /><i /></span>
              </p>
            )}
            {m.kind === 'result' && <ResultCard result={m.result} />}
          </div>
        ))}
        <div ref={endRef} />
      </div>

      <div className="composer">
        <AudioRecorder onRecorded={identify} disabled={busy} />
        <button
          type="button"
          className="upload-btn"
          disabled={busy}
          onClick={() => fileRef.current?.click()}
        >
          Upload audio
        </button>
        <input
          ref={fileRef}
          type="file"
          accept="audio/*,.wav,.mp3,.webm,.ogg,.m4a"
          hidden
          data-testid="file-input"
          onChange={(e) => { onFiles(e.target.files); e.target.value = '' }}
        />
        <span className="hint">or drop a file anywhere in this window</span>
      </div>
      {dragging && <div className="drop-overlay">Drop audio to identify</div>}
    </section>
  )
}
