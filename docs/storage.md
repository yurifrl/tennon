# Storage Backends

Every MCPHub object is a CRD. The CRD **schema** is fixed; where those objects are
**persisted** is pluggable. The controller talks to its store through one storage interface
with three drivers. Switching backends changes *where* objects live, not *what* they are —
`kubectl`-applied YAML and UI-created objects behave identically on any backend.

## The three backends

### Kubernetes / etcd (default)

CRDs stored in the cluster's etcd via the Kubernetes API. The controller uses
controller-runtime watches/informers directly. You get native `kubectl` and GitOps
workflows, Kubernetes RBAC, and watch-based reconciliation. Best when running inside a
cluster with a team.

### Postgres

CRD objects stored as rows in Postgres. The controller runs the **same** CRD types and
reconcile loops against an embedded object store that emulates the apiserver's
get/list/watch semantics. Best when running outside a cluster or wanting a shared/HA
external database.

### SQLite

CRD objects in a local SQLite file. Same types and loops as above, single-node. Best for one
person, local development, or an embedded self-host deployment.

## Why this design

- **Solo to team (R1).** SQLite covers one person; Postgres or Kubernetes covers a team — no
  product fork.
- **Consistency.** Because the types and reconcile logic are identical, the UI, CLI, and
  GitOps all behave the same regardless of backend.
- **UI ↔ CRD parity.** "Create MCP" in the UI is an API call that writes an `Mcp` object into
  whichever backend is configured; the same object can be created with `kubectl apply` on the
  Kubernetes backend or seeded via SQL/migration on the others.

## Secrets per backend

- **Kubernetes:** native `Secret` objects, referenced by `secretKeyRef`.
- **Postgres / SQLite:** an encrypted store-native secret table with the same reference
  shape.

Either way, secret values are rendered into the upstream MCP at call time and never appear in
API responses, status, logs, or the UI. See [Security](security.md).

## Choosing and switching

Set the backend at startup (`--store=kubernetes|postgres|sqlite`). Migrating between backends
is an export/import of the CRD objects; runtime state (caches, routing) is rebuilt by
reconciliation and does not need migrating.
