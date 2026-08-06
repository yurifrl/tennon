import { useEffect, useState, FormEvent } from 'react'
import { Secret, SecretCreateRequest } from '../types'

export default function Secrets() {
  const [secrets, setSecrets] = useState<Secret[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [secretName, setSecretName] = useState('')
  const [keyValuePairs, setKeyValuePairs] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    fetchSecrets()
  }, [])

  const fetchSecrets = async () => {
    try {
      const res = await fetch('/api/v1/secrets')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setSecrets(data.items || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitting(true)

    // Parse key=value pairs
    const data: Record<string, string> = {}
    const lines = keyValuePairs.split('\n').filter((l) => l.trim())
    for (const line of lines) {
      const [key, ...valueParts] = line.split('=')
      if (key && valueParts.length > 0) {
        data[key.trim()] = valueParts.join('=').trim()
      }
    }

    const payload: SecretCreateRequest = { name: secretName, data }

    try {
      const res = await fetch('/api/v1/secrets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(`HTTP ${res.status}: ${text}`)
      }
      alert('Secret created')
      setShowCreate(false)
      setSecretName('')
      setKeyValuePairs('')
      await fetchSecrets()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to create')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <div className="container">Loading...</div>
  if (error) return <div className="container">Error: {error}</div>

  return (
    <div className="container">
      <div className="flex items-center gap-2 mb-4">
        <h1 style={{ fontSize: '24px', fontWeight: 600, margin: 0, flex: 1 }}>Secrets</h1>
        {!showCreate && (
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
            + Create Secret
          </button>
        )}
      </div>
      <p className="text-muted mb-4">
        Kubernetes Secrets, by convention named <span className="mono">tennon-&lt;connector&gt;</span>.
        Values are never shown.
      </p>

      {showCreate && (
        <div className="card mb-4">
          <div className="card-header">
            <h3 className="card-title">Create Secret</h3>
          </div>
          <div className="card-body">
            <form onSubmit={handleCreate}>
              <div className="form-group">
                <label className="form-label">Name</label>
                <input
                  type="text"
                  className="form-input"
                  value={secretName}
                  onChange={(e) => setSecretName(e.target.value)}
                  required
                  placeholder="e.g. tennon-github"
                />
              </div>

              <div className="form-group">
                <label className="form-label">
                  Key-Value Pairs <span className="text-muted text-xs">(one per line: key=value)</span>
                </label>
                <textarea
                  className="form-input"
                  style={{ fontFamily: 'monospace', minHeight: '120px' }}
                  value={keyValuePairs}
                  onChange={(e) => setKeyValuePairs(e.target.value)}
                  required
                  placeholder="GITHUB_TOKEN=ghp_abc123&#10;API_KEY=secret123"
                />
              </div>

              <div className="flex gap-2 justify-end">
                <button
                  type="button"
                  className="btn"
                  onClick={() => {
                    setShowCreate(false)
                    setSecretName('')
                    setKeyValuePairs('')
                  }}
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={submitting}>
                  {submitting ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="card">
        <div className="table-container" style={{ overflowX: 'auto' }}>
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Keys</th>
              </tr>
            </thead>
            <tbody>
              {secrets.length === 0 ? (
                <tr>
                  <td colSpan={2} style={{ textAlign: 'center', color: '#6b7587' }}>
                    No secrets yet.
                  </td>
                </tr>
              ) : (
                secrets.map((s) => (
                  <tr key={s.name}>
                    <td>
                      <strong className="mono">{s.name}</strong>
                    </td>
                    <td className="text-muted text-xs">
                      {s.keys.join(', ') || 'none'}
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
