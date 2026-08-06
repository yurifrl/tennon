// Package web serves Tennon's embedded web UI. Phase 5 replaces this placeholder
// with the built Vite/React assets embedded via go:embed; for now it serves a
// minimal landing page so the single-binary surface is complete.
package web

import "net/http"

const indexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>Tennon</title>
<style>body{font-family:system-ui,sans-serif;margin:3rem auto;max-width:40rem;color:#222}
code{background:#f4f4f4;padding:.1rem .3rem;border-radius:.2rem}</style></head>
<body>
<h1>Tennon</h1>
<p>The Kubernetes-native control plane and proxy for Model Context Protocol servers.</p>
<p>The click-through web UI ships in roadmap phase 5. Until then, use the API and CLI:</p>
<ul>
<li><code>GET /api/v1/connectors</code> — list connectors</li>
<li><code>GET /healthz</code> — health</li>
<li><code>tctl connector add … / list / describe</code></li>
</ul>
<p>Connectors are proxied on the proxy port at <code>/mcp/&lt;name&gt;</code>.</p>
</body></html>`

// Handler returns the embedded web UI handler.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})
	return mux
}
