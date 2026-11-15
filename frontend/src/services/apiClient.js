// Thin fetch wrapper around the Go API. All calls throw Error(message) on failure.

async function parse(res) {
  let body = null
  try {
    body = await res.json()
  } catch {
    /* non-JSON error body */
  }
  if (!res.ok || !body || body.status !== 'success') {
    throw new Error(body?.message || `Request failed (${res.status})`)
  }
  return body.data
}

/** POST /api/v1/recognize — resolves to { matched, confidence, votes, match_offset_seconds, song }. */
export async function recognizeAudio(blob, filename = 'clip.webm') {
  const form = new FormData()
  form.append('audio', blob, filename)
  return parse(await fetch('/api/v1/recognize', { method: 'POST', body: form }))
}

/** POST /api/v1/songs — fields: title, artist, album, release_date, cover_art_url, spotify, youtube. */
export async function ingestSong(file, fields) {
  const form = new FormData()
  form.append('audio', file, file.name)
  for (const [k, v] of Object.entries(fields)) {
    if (v) form.append(k, v)
  }
  return parse(await fetch('/api/v1/songs', { method: 'POST', body: form }))
}

/** GET /api/v1/songs */
export async function listSongs() {
  const data = await parse(await fetch('/api/v1/songs'))
  return data.songs
}
