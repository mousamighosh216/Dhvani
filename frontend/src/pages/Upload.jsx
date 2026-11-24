import { AddSong } from './Library.jsx'
import { Crumbs } from '../ui.jsx'

export default function Upload() {
  return (
    <>
      <Crumbs here="Upload a Song" />
      <div className="wrap narrow2 section">
        <h1 className="h2s">Add a song to Dhvani</h1>
        <p className="muted">Share the track and its details so it can be identified in the catalog.</p>
        <AddSong />
      </div>
    </>
  )
}
