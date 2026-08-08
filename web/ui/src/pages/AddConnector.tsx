import { useState, FormEvent } from 'react'

export default function AddConnector() {
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [transport, setTransport] = useState('http')
  const [exposePrefix, setExposePrefix] = useState('')
  const [allow, setAllow] = useState('')
  const [tags, setTags] = useState('')
  const [headerKey, setHeaderKey] = useState('')
  const [headerValue, setHeaderValue] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitting(true)

    const payload = {
      kind: 'Connector',
      metadata: { name },
      spec: {
        runtime: {
          remote: { url },
        },
        transport,
        tools: {
          exposePrefix: exposePrefix || undefined,
          allow: allow
            ? allow
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean)
            : undefined,
        },
        headers: headerKey && headerValue ? { [headerKey]: headerValue } : undefined,
        tags: tags
          ? tags
              .split(',')
              .map((s) => s.trim())
              .filter(Boolean)
          : undefined,
      },
    }

    try {
      const res = await fetch('/api/v1/connectors', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(`HTTP ${res.status}: ${text}`)
      }
      alert('Connector created')
      window.location.hash = '#/'
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to create')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="container">
      <h1 style={{ fontSize: '24px', fontWeight: 600, marginBottom: '8px' }}>Add Connector</h1>
      <p className="text-muted mb-4">
        Add a remote MCP connector. Specify the URL, transport, exposure rules, and tags.
      </p>

      <div className="card">
        <div className="card-body">
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Name</label>
              <input
                type="text"
                className="form-input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                placeholder="e.g. github"
              />
            </div>

            <div className="form-group">
              <label className="form-label">Remote URL</label>
              <input
                type="url"
                className="form-input"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                required
                placeholder="https://example.com/mcp"
              />
            </div>

            <div className="form-group">
              <label className="form-label">Transport</label>
              <select
                className="form-select"
                value={transport}
                onChange={(e) => setTransport(e.target.value)}
              >
                <option value="http">http</option>
                <option value="sse">sse</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">
                Expose Prefix <span className="text-muted text-xs">(optional)</span>
              </label>
              <input
                type="text"
                className="form-input mono"
                value={exposePrefix}
                onChange={(e) => setExposePrefix(e.target.value)}
                placeholder="e.g. github_"
              />
            </div>

            <div className="form-group">
              <label className="form-label">
                Allow Globs <span className="text-muted text-xs">(comma-separated, optional)</span>
              </label>
              <input
                type="text"
                className="form-input mono"
                value={allow}
                onChange={(e) => setAllow(e.target.value)}
                placeholder="e.g. search_*, get_*"
              />
            </div>

            <div className="form-group">
              <label className="form-label">
                Tags <span className="text-muted text-xs">(comma-separated, optional)</span>
              </label>
              <input
                type="text"
                className="form-input"
                value={tags}
                onChange={(e) => setTags(e.target.value)}
                placeholder="e.g. code, prod"
              />
            </div>

            <div className="form-group">
              <label className="form-label">
                Credential Header <span className="text-muted text-xs">(optional)</span>
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  className="form-input"
                  style={{ flex: 1 }}
                  value={headerKey}
                  onChange={(e) => setHeaderKey(e.target.value)}
                  placeholder="Authorization"
                />
                <input
                  type="text"
                  className="form-input mono"
                  style={{ flex: 2 }}
                  value={headerValue}
                  onChange={(e) => setHeaderValue(e.target.value)}
                  placeholder="Bearer {{secret:github-token}}"
                />
              </div>
              <p className="text-muted text-xs mt-2">
                Use <span className="mono">{'{{secret:name}}'}</span> to reference a secret.
              </p>
            </div>

            <div className="flex gap-2 justify-end">
              <a href="#/" className="btn" style={{ textDecoration: 'none' }}>
                Cancel
              </a>
              <button type="submit" className="btn btn-primary" disabled={submitting}>
                {submitting ? 'Creating...' : 'Create Connector'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
