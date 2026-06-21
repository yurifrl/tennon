# The `Mcp` Manifest & State-Backed Configuration

MCPHub does **not** use a single `mcp.yaml` / `config.yaml` as its source of truth. Instead,
every MCP server is declared as an `Mcp` custom resource. The controller reconciles those
resources into runtime state, and the proxy serves from that state.

This is the central twist versus MCP Shield: **configuration is Kubernetes-native and
stored in state, not in one file.**

## Why state-backed?

- **Multiple authors.** MCPs can be created via `kubectl`, GitOps, the API, or the web UI.
  There is no single file to serialize edits through.
- **Live updates.** Adding or changing an `Mcp` triggers reconciliation; no restart and no
  file reload.
- **Secret safety.** Secret material is referenced, never inlined, and is rendered only
  inside the cluster at spawn time.

## The `Mcp` kind

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Mcp
metadata:
  name: github
  namespace: mcphub
spec:
  # How to run the upstream MCP server.
  runtime:
    type: container          # container | remote
    image: ghcr.io/github/github-mcp-server:latest
    command: ["server"]
    args: ["--stdio"]

  # Transport the proxy uses to reach this server.
  transport: stdio           # stdio | streamable-http | sse

  # Secret rendering: referenced, never inlined.
  env:
    - name: GITHUB_PERSONAL_ACCESS_TOKEN
      valueFrom:
        secretKeyRef:
          name: github-credentials
          key: token

  # Optional headers (for remote transports) can also render from secrets.
  headers:
    - name: Authorization
      valueFrom:
        secretKeyRef:
          name: github-credentials
          key: bearer
          format: "Bearer {{ .value }}"

  # Tool surface controls.
  tools:
    exposePrefix: "github_"  # how proxied tool names appear to clients
    allow: ["search_*", "get_*", "list_*"]
    deny: ["delete_*"]
status:
  phase: Ready               # Pending | Ready | Degraded | Failed
  url: https://mcphub/internal/mcp/github
  discoveredTools: 42
  exposedTools: 18
  lastReconciled: "2026-06-19T12:00:00Z"
```

### A remote MCP example

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Mcp
metadata:
  name: weather
spec:
  runtime:
    type: remote
    url: https://weather-mcp.example.com/sse
  transport: sse
  tools:
    exposePrefix: "weather_"
```

## Secret rendering

The controller is responsible for turning secret references into the values the upstream
server needs — and for keeping those values inside the cluster.

1. The `Mcp` spec references `Secret`s via `secretKeyRef` (never literal values).
2. On reconcile/spawn, the controller resolves the `Secret` and renders it into:
   - **env vars** for container-runtime MCPs, or
   - **request headers** for remote-transport MCPs.
3. An optional `format` template (e.g. `"Bearer {{ .value }}"`) wraps the rendered value.
4. Rendered values live only in the spawned workload's environment and the proxy's
   in-memory cache. They are **never** written back to the `Mcp` status, the API
   responses, the catalog, or logs.

```mermaid
sequenceDiagram
    participant CR as Mcp CR
    participant Ctrl as Controller
    participant Sec as K8s Secret
    participant Up as Upstream MCP server

    Ctrl->>CR: Reconcile Mcp
    Ctrl->>Sec: Resolve secretKeyRef
    Sec->>Ctrl: Secret value (in-cluster only)
    Ctrl->>Ctrl: Apply format template
    Ctrl->>Up: Spawn with rendered env / headers
    Note over Ctrl,Up: Rendered values never leave the cluster
```

## Related resources

- **`McpBundle`** — composes tools from one or more `Mcp`s into a single endpoint. See
  [bundles.md](bundles.md).
- **`McpPolicy`** — authorization rules mapping subjects (OAuth2 identities, service
  tokens) to allowed MCPs/tools. See [authentication-flow.md](authentication-flow.md).
