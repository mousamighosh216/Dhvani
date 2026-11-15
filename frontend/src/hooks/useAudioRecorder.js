import { useCallback, useEffect, useRef, useState } from 'react'

export const MAX_RECORD_SECONDS = 10

/**
 * Records the microphone with MediaRecorder (WebM/Opus in Chrome/Firefox).
 * The backend converts it with ffmpeg. Calls onComplete(blob, filename) when
 * recording stops, either manually or at MAX_RECORD_SECONDS.
 */
export function useAudioRecorder(onComplete) {
  const [recording, setRecording] = useState(false)
  const [seconds, setSeconds] = useState(0)
  const [level, setLevel] = useState(0) // 0..1 live input level
  const [error, setError] = useState(null)
  const refs = useRef({})
  const onCompleteRef = useRef(onComplete)
  onCompleteRef.current = onComplete

  const cleanup = useCallback(() => {
    const r = refs.current
    clearInterval(r.timer)
    cancelAnimationFrame(r.raf)
    r.stream?.getTracks().forEach((t) => t.stop())
    r.audioCtx?.close().catch(() => {})
    refs.current = {}
    setRecording(false)
    setLevel(0)
  }, [])

  const stop = useCallback(() => {
    const rec = refs.current.recorder
    if (rec && rec.state !== 'inactive') rec.stop()
  }, [])

  const start = useCallback(async () => {
    setError(null)
    if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') {
      setError('Microphone recording is not supported in this browser.')
      return
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const recorder = new MediaRecorder(stream)
      const chunks = []
      recorder.ondataavailable = (e) => e.data.size > 0 && chunks.push(e.data)
      recorder.onstop = () => {
        const type = recorder.mimeType || 'audio/webm'
        const ext = type.includes('mp4') ? 'm4a' : type.includes('ogg') ? 'ogg' : 'webm'
        cleanup()
        onCompleteRef.current?.(new Blob(chunks, { type }), `recording.${ext}`)
      }

      // Live level meter.
      const audioCtx = new (window.AudioContext || window.webkitAudioContext)()
      const analyser = audioCtx.createAnalyser()
      analyser.fftSize = 512
      audioCtx.createMediaStreamSource(stream).connect(analyser)
      const buf = new Uint8Array(analyser.fftSize)
      const tick = () => {
        analyser.getByteTimeDomainData(buf)
        let peak = 0
        for (const v of buf) peak = Math.max(peak, Math.abs(v - 128))
        setLevel(Math.min(1, peak / 90))
        refs.current.raf = requestAnimationFrame(tick)
      }

      refs.current = { stream, recorder, audioCtx }
      recorder.start()
      setRecording(true)
      setSeconds(0)
      tick()
      const startedAt = Date.now()
      refs.current.timer = setInterval(() => {
        const s = Math.floor((Date.now() - startedAt) / 1000)
        setSeconds(s)
        if (s >= MAX_RECORD_SECONDS) stop()
      }, 250)
    } catch (e) {
      cleanup()
      setError(
        e?.name === 'NotAllowedError'
          ? 'Microphone permission was denied.'
          : `Could not start recording: ${e?.message || e}`,
      )
    }
  }, [cleanup, stop])

  useEffect(() => () => cleanup(), [cleanup])

  return { recording, seconds, level, error, start, stop }
}
