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
- **UI ↔ CRD parity.** "Create Connector" in the UI is an API call that writes an `McpConnector` object into
  whichever backend is configured; the same object can be created with `kubectl apply` on the
  Kubernetes backend or seeded via SQL/migration on the others.

## One store, including audit — no separate events database

Teleport splits storage in two: a **cluster-state backend** (users, roles, config) and a
separate **audit-events backend**, because config is transactional while events are
append-heavy time-series — `etcd`, for instance, can hold Teleport's state but not its audit
log.

MCPHub deliberately **does not** do this. There is **one** configured store, and it holds
*everything*: the desired-state CRD objects **and** the audit/event records. Whichever backend
you pick — Kubernetes, Postgres, or SQLite — is the single source of truth for both.

Why diverge from Teleport here:

- **Solo-to-team simplicity (R1).** A second database to provision, back up, and migrate
  defeats the "one binary, one store" goal. One person on SQLite gets audit for free.
- **Volume fits.** MCPHub audits tool calls and policy decisions, not high-frequency session
  streams; a single relational/etcd-backed store handles that load without a time-series tier.
- **Consistency.** Config and its audit trail share one transactional store, so an action and
  the event recording it cannot diverge across two systems.

> **ponytail:** single store until audit volume measurably strains it. If it ever does, the
> upgrade path is Teleport's: split the events table onto its own backend behind the same
> storage interface — not before.

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
