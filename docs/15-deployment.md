# Deployment

MCPHub ships as one server binary — `mcphub` (controller + API + MCP proxy + embedded web UI)
— distributed as a container image, a Helm chart, and a released install bundle. The
`mcphubctl` CLI is a separate client binary. Pick a [storage backend](storage.md) to match
your scale.

## Solo / local (SQLite)

Run the single binary directly; the UI is served on the API port.

```bash
mcphub \
  --store=sqlite --store-path=./mcphub.db \
  --api-addr=:8080 --proxy-addr=:8081 \
  --oidc-issuer=https://accounts.google.com --oidc-client-id=mcphub
```

No cluster required. Good for one person or evaluation.

## Single host / team (Postgres)

Point the same binary at Postgres for shared, HA-capable state, still without Kubernetes:

```bash
mcphub --store=postgres --store-dsn="postgres://…/mcphub" \
  --api-addr=:8080 --proxy-addr=:8081 …
```

Front it with your reverse proxy / ingress as needed.

## Kubernetes (team, native)

Install with Helm — the primary path:

```bash
helm install mcphub oci://ghcr.io/acme/charts/mcphub -n mcphub --create-namespace \
  --set oidc.issuer=https://accounts.google.com --set oidc.clientId=mcphub
```

For Postgres, point the chart at a Secret holding the DSN (keeps the password out of process
arguments):

```bash
kubectl -n mcphub create secret generic mcphub-store --from-literal=dsn="postgres://…/mcphub"
helm upgrade mcphub oci://ghcr.io/acme/charts/mcphub -n mcphub \
  --set store.backend=postgres --set store.existingSecret=mcphub-store
```

The controller installs its own CRDs on first start, so there is no separate CRD step. If you
manage cluster state declaratively (GitOps), apply the released bundle (CRDs + RBAC +
controller in one manifest) instead of Helm:

```bash
kubectl apply -f https://github.com/acme/mcphub/releases/latest/download/mcphub.yaml
```

On Kubernetes you get native `kubectl`/GitOps management of `McpConnector`/`Policy`,
Kubernetes RBAC for the control plane, and watch-based reconciliation. Container-runtime
connectors are scheduled as workloads. See [CRD Lifecycle](crd-lifecycle.md) for how CRDs are
delivered and versioned.

## Configuration essentials

- **Identity providers:** OIDC issuer(s) and client IDs for Google and/or Okta; configure
  machine trust relationships (OIDC/K8s/SPIFFE/cloud) for tokenless auth.
- **Secrets:** Kubernetes `Secret`s (Kubernetes backend) or an encryption key for the
  store-native secret table (Postgres/SQLite).
- **Networking:** expose the proxy endpoint and the API (which also serves the UI); keep the
  API internal where possible.
- **TTLs & limits:** default credential TTL (~1h) and org maximum token lifetime.

## Agents

Agents connect **directly** to `/mcp/<connector>` over Streamable HTTP, or — for clients that
launch a stdio command — via the CLI's MCP server (`mcphubctl mcp serve --transport stdio`).
See [Getting Started](getting-started.md#4-connect-an-agent).

## Upgrades

Roll the controller image (Helm `--set image.tag=…` or re-apply the bundle); runtime state is
rebuilt by reconciliation and desired state persists in the configured backend. CRD schema
changes are versioned under `mcphub.io/v1alpha1` — see [CRD Lifecycle](crd-lifecycle.md).
