import { useEffect, useState } from 'react'
import { Connector } from '../types'

export default function Connectors() {
  const [connectors, setConnectors] = useState<Connector[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchConnectors()
  }, [])

  const fetchConnectors = async () => {
    try {
      const res = await fetch('/api/v1/connectors')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setConnectors(data.items || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete connector "${name}"?`)) return
    try {
      const res = await fetch(`/api/v1/connectors/${name}`, { method: 'DELETE' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      await fetchConnectors()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  if (loading) return <div className="container">Loading...</div>
  if (error) return <div className="container">Error: {error}</div>

  return (
    <div className="container">
      <div className="flex items-center gap-2 mb-4">
        <h1 style={{ fontSize: '24px', fontWeight: 600, margin: 0, flex: 1 }}>Connectors</h1>
        <a href="#/add" className="btn btn-primary" style={{ textDecoration: 'none' }}>
          + Add Connector
        </a>
      </div>
      <p className="text-muted mb-4">
        Tool providers in this workspace — MCP servers (remote), OpenAPI &amp; GraphQL services.
      </p>

      <div className="card">
        <div className="table-container" style={{ overflowX: 'auto' }}>
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Kind</th>
                <th>Runtime</th>
                <th>URL</th>
                <th>Phase</th>
                <th>Tools</th>
                <th>Tags</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {connectors.length === 0 ? (
                <tr>
                  <td colSpan={8} style={{ textAlign: 'center', color: '#6b7587' }}>
                    No connectors yet. Add one to get started.
                  </td>
                </tr>
              ) : (
                connectors.map((c) => (
                  <tr key={c.metadata.name}>
                    <td>
                      <strong>{c.metadata.name}</strong>
                    </td>
                    <td>
                      <span className="tag tag-blue">{c.kind}</span>
                    </td>
                    <td>
                      <span className="tag tag-green">remote</span>
                    </td>
                    <td className="mono text-xs text-muted">
                      {c.spec.runtime.remote?.url || '—'}
                    </td>
                    <td>
                      <span
                        className={`tag ${
                          c.status?.phase === 'Ready' ? 'tag-green' : 'tag-amber'
                        }`}
                      >
                        {c.status?.phase || 'Unknown'}
                      </span>
                    </td>
                    <td>{c.status?.discoveredTools || 0}</td>
                    <td>
                      {c.spec.tags?.map((tag) => (
                        <span key={tag} className="tag" style={{ marginRight: '4px' }}>
                          {tag}
                        </span>
                      ))}
                    </td>
                    <td>
                      <button
                        className="btn"
                        style={{ fontSize: '11px', color: '#f25c5c', borderColor: '#4a2a2f' }}
                        onClick={() => handleDelete(c.metadata.name)}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
