function formatTime(sec) {
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

export default function ResultCard({ result }) {
  const { song, confidence = 0, votes, match_offset_seconds: offset = 0 } = result
  const pct = Math.round(confidence * 100)
  const links = song.external_links || {}

  return (
    <article className="result-card" aria-label={`Identified ${song.title}`}>
      <div className="cover">
        {song.album?.cover_art_url ? (
          <img src={song.album.cover_art_url} alt={`${song.album.title} cover`} />
        ) : (
          <span aria-hidden="true">♪</span>
        )}
      </div>
      <div className="meta">
        <h3>{song.title}</h3>
        <p className="artist">{song.artist?.name}</p>
        {song.album && (
          <p className="album">
            {song.album.title}
            {song.album.release_date ? ` · ${song.album.release_date.slice(0, 4)}` : ''}
          </p>
        )}
        <div className="confidence" title="Share of matching landmarks that agree on one alignment">
          <div className="bar"><div style={{ width: `${Math.max(pct, 4)}%` }} /></div>
          <span>{pct}% match{votes ? ` · ${votes} landmarks` : ''}</span>
        </div>
        <p className="offset">Your clip starts at {formatTime(offset)} of {formatTime(song.duration_seconds)}</p>
        {(links.spotify || links.youtube) && (
          <div className="links">
            {links.spotify && <a href={links.spotify} target="_blank" rel="noreferrer">Spotify</a>}
            {links.youtube && <a href={links.youtube} target="_blank" rel="noreferrer">YouTube</a>}
          </div>
        )}
      </div>
    </article>
  )
}
