# Deployment

MCPHub runs as `mcphub-controller` (controller + API + proxy) and `mcphub-web` (UI). Choose a
[storage backend](storage.md) to match your scale.

## Solo / local (SQLite)

```bash
mcphub-controller \
  --store=sqlite --store-path=./mcphub.db \
  --api-addr=:8080 --proxy-addr=:8081 \
  --oidc-issuer=https://accounts.google.com --oidc-client-id=mcphub

mcphub-web --listen=:3000 --api-url=http://localhost:8080 \
  --oidc-issuer=https://accounts.google.com --oidc-client-id=mcphub
```

No cluster required. Good for one person or evaluation.

## Single host / team (Postgres)

Point the controller at Postgres for shared, HA-capable state, still without Kubernetes:

```bash
mcphub-controller --store=postgres --store-dsn="postgres://…/mcphub" …
```

Run `mcphub-web` alongside (or behind your reverse proxy / ingress).

## Kubernetes (team, native)

```bash
make install                       # apply CRDs
make deploy IMG=<registry>/mcphub-controller:tag
# deploy mcphub-web similarly, expose via Ingress/Gateway
```

On Kubernetes you get native `kubectl`/GitOps management of `McpConnector`/`Policy`,
Kubernetes RBAC for the control plane, and watch-based reconciliation. Container-runtime
connectors are scheduled as workloads.

## Configuration essentials

- **Identity providers:** OIDC issuer(s) and client IDs for Google and/or Okta; configure
  machine trust relationships (OIDC/K8s/SPIFFE/cloud) for tokenless auth.
- **Secrets:** Kubernetes `Secret`s (Kubernetes backend) or an encryption key for the
  store-native secret table (Postgres/SQLite).
- **Networking:** expose `mcphub-web` and the proxy endpoint; keep the controller API
  internal where possible.
- **TTLs & limits:** default credential TTL (~1h) and org maximum token lifetime.

## Agents

Agents connect **directly** to `/mcp/<connector>` over Streamable HTTP,
or — for clients that launch a stdio command — via the CLI's MCP server (`mcphub mcp serve
--transport stdio`). See [Getting Started](getting-started.md#4-connect-an-agent).

## Upgrades

CRD schema changes are versioned under `mcphub.io/v1alpha1`. Runtime state is rebuilt by
reconciliation, so rolling the controller is safe; desired state persists in the configured
backend.
