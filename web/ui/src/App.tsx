import { useState, useEffect } from 'react'
import Connectors from './pages/Connectors'
import AddConnector from './pages/AddConnector'
import Secrets from './pages/Secrets'
import './styles.css'

type Page = 'connectors' | 'add' | 'secrets'

function App() {
  const [page, setPage] = useState<Page>('connectors')

  useEffect(() => {
    const handleHashChange = () => {
      const hash = window.location.hash.slice(1) // Remove #
      if (hash === '/add') setPage('add')
      else if (hash === '/secrets') setPage('secrets')
      else setPage('connectors')
    }
    handleHashChange()
    window.addEventListener('hashchange', handleHashChange)
    return () => window.removeEventListener('hashchange', handleHashChange)
  }, [])

  return (
    <>
      <header className="header">
        <div
          style={{
            width: '28px',
            height: '28px',
            borderRadius: '8px',
            background: 'linear-gradient(135deg, #00c29f, #4c8dff)',
            display: 'grid',
            placeItems: 'center',
            fontWeight: 800,
            fontSize: '16px',
            color: '#04121a',
          }}
        >
          M
        </div>
        <h1 className="header-title">Tennon</h1>
      </header>

      <nav className="nav">
        <a
          href="#/"
          className={`nav-link ${page === 'connectors' ? 'active' : ''}`}
          onClick={(e) => {
            e.preventDefault()
            window.location.hash = '#/'
          }}
        >
          Connectors
        </a>
        <a
          href="#/secrets"
          className={`nav-link ${page === 'secrets' ? 'active' : ''}`}
          onClick={(e) => {
            e.preventDefault()
            window.location.hash = '#/secrets'
          }}
        >
          Secrets
        </a>
      </nav>

      <main style={{ paddingTop: '20px' }}>
        {page === 'connectors' && <Connectors />}
        {page === 'add' && <AddConnector />}
        {page === 'secrets' && <Secrets />}
      </main>
    </>
  )
}

export default App
