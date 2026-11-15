import { useState } from 'react'
import ChatWindow from './components/ChatWindow.jsx'
import AddSongPanel from './components/AddSongPanel.jsx'

export default function App() {
  const [panelOpen, setPanelOpen] = useState(false)
  return (
    <div className={`app ${panelOpen ? 'with-panel' : ''}`}>
      <header className="topbar">
        <div className="brand">
          <svg viewBox="0 0 32 32" width="26" height="26" aria-hidden="true">
            <rect width="32" height="32" rx="8" fill="var(--accent)" />
            <path d="M7 16h2M12 11v10M17 7v18M22 12v8M26 15v2" stroke="#062b27" strokeWidth="2.4" strokeLinecap="round" />
          </svg>
          <span>Dhwani</span>
        </div>
        <button type="button" className="ghost" onClick={() => setPanelOpen((o) => !o)}>
          {panelOpen ? 'Hide catalog' : 'Catalog'}
        </button>
      </header>
      <main>
        <ChatWindow />
        {panelOpen && <AddSongPanel onClose={() => setPanelOpen(false)} />}
      </main>
    </div>
  )
}
