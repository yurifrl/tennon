# Project Structure

How the MCPHub codebase is laid out, and the one rule that keeps it honest:
**packages only call down — never up, never sideways.**

This page is organized like every design page in these docs:

- **Spec** — what the structure must guarantee (the rules).
- **Design** — how the packages are arranged to meet them.
- **Usage** — how you work within the layout day to day.

---

## Spec

The structure exists to enforce these invariants. They are not style preferences — a build
that violates them should fail.

1. **Layered, acyclic dependencies.** Packages are assigned a layer. A package may import
   only packages in a **lower** layer. No upward imports, no sibling imports inside the same
   layer, no cycles.
2. **The domain has no infrastructure.** The core (`domain`) imports nothing from
   Kubernetes, the store, the HTTP stack, the proxy, or any vendor SDK. It is pure Go types
   and interfaces.
3. **Dependencies point inward via interfaces.** When a lower layer needs behavior that
   lives higher up, it declares an **interface** it owns; the higher layer implements it.
   This is the only way control flows "up" — through an abstraction the callee defines.
4. **One composition root.** Concrete implementations are wired together in exactly one
   place (`cmd/`), using [Uber FX](technologies.md#dependency-injection--uber-fx). No package
   constructs its own dependencies from global state.
5. **Secrets never cross a boundary they shouldn't.** No credential value appears in a type
   that leaves the controller (API responses, logs, status). The structure makes this
   reviewable: only `secret` and `proxy` touch rendered secrets.

### Success criteria

- `go list` import graph is acyclic (enforced in CI).
- `domain` imports zero third-party/infra packages (enforced by an import-lint allowlist).
- Every concrete dependency is provided in `cmd/`, not `init()` or package-level singletons.

---

## Design

### The dependency rule

> A package may depend only on packages **below** it. Lower layers know nothing about higher
> layers except through interfaces the lower layer defines.

Reading the diagram top-to-bottom, every arrow points **down**:

```
                 cmd/            ← composition root (FX wiring, main)
                   │
        ┌──────────┼───────────┬──────────────┐
        ▼          ▼           ▼              ▼
      api        proxy      controller       cli      ← delivery / adapters
        └──────────┴───────────┴──────────────┘
                          │
            ┌─────────────┼──────────────┐
            ▼             ▼              ▼
         store        runtime         secret          ← infrastructure
            └─────────────┴──────────────┘
                          │
                          ▼
                       domain                          ← core (no imports out)
```

- **`domain` (bottom).** Types and rules: `Connector` (`McpConnector`, `OpenApiConnector`,
  `GraphqlConnector`), `Policy`,
  `Token`, `ServiceAccount`, `Subject`, authorization decisions. Also the **port
  interfaces** the upper layers must satisfy (`Store`, `Runtime`, `SecretRenderer`,
  `IdentityVerifier`). Imports nothing outward.
- **Infrastructure (`store`, `runtime`, `secret`).** Concrete implementations of domain
  ports. `store` persists CRD objects (etcd/Postgres/SQLite — see
  [Storage backends](storage.md)). `runtime` starts/stops MCP workloads (Docker or Pods —
  see [Runtime & Isolation](runtime-isolation.md)). `secret` resolves and renders
  credentials. Each imports `domain` only.
- **Delivery / adapters (`api`, `proxy`, `controller`, `cli`).** Translate the outside world
  into domain calls. `api` is REST/JSON; `proxy` is the MCP data plane; `controller`
  reconciles CRDs ([Kubebuilder](technologies.md#operator-framework--kubebuilder)); `cli` is
  the client + embedded MCP server. They import `domain` and the infrastructure ports — never
  each other.
- **`cmd/` (top).** `main` per binary. Builds the FX graph: chooses which `Store`/`Runtime`
  implementation to provide, wires `api`/`proxy`/`controller`, starts the process. This is
  the **only** layer allowed to know every concrete type.

### How "up" calls work (without breaking the rule)

The controller needs to react to store changes; the proxy needs an authorization decision.
Neither imports the layer that drives it. Instead, the **lower** layer owns the interface:

```go
// package domain — the port lives with the core, not with the caller
type Authorizer interface {
    Decide(ctx context.Context, sub Subject, req ToolCall) Decision
}
```

`proxy` depends on `domain.Authorizer`; the concrete authorizer is implemented in a lower
layer and **provided in `cmd/`**. Control appears to flow up; the *import* still points down.
This is the inversion that keeps the graph acyclic.

### Proposed package tree

```
mcphub/
├── cmd/
│   ├── mcphub/            # server binary: controller + api + proxy + ui
│   └── mcphubctl/         # cli binary (also `mcp serve` bridge)
├── domain/               # core types + port interfaces (no outward imports)
│   ├── connector.go
│   ├── policy.go
│   ├── identity.go       # Subject, Token, ServiceAccount
│   └── ports.go          # Store, Runtime, SecretRenderer, Authorizer, IdentityVerifier
├── store/                # domain.Store impls: etcd, postgres, sqlite
├── runtime/              # domain.Runtime impls: docker, kubernetes
├── secret/               # domain.SecretRenderer impl
├── auth/                 # identity verifiers (OIDC/Okta, K8s SA, SPIFFE) + token minting
├── controller/           # Kubebuilder reconcilers
├── api/                  # REST handlers
├── proxy/                # MCP streamable-HTTP data plane
├── cli/                  # CLI commands + embedded MCP server
└── web/                  # embedded Vite/React assets + thin Go server
```

`auth` sits in the infrastructure layer (it implements `domain.IdentityVerifier`); `api`,
`proxy`, and `controller` consume it, never the reverse.

---

## Usage

### Adding a feature — pick the layer first

1. **New rule or object?** Add the type/interface to `domain`. Stop and check nothing in
   `domain` now imports outward.
2. **New backend or adapter?** Implement an existing `domain` port in `store`/`runtime`/
   `secret`/`auth`. Provide it in `cmd/`.
3. **New endpoint or tool surface?** Add it in `api`/`proxy`/`cli`, calling **down** into
   domain ports. Do not reach into another delivery package.
4. **Wiring?** Register providers in `cmd/`. If you find yourself constructing a dependency
   anywhere else, move it up to the composition root.

### What the rule forbids (and the fix)

| Smell | Why it's wrong | Fix |
|-------|----------------|-----|
| `proxy` imports `api` | sideways | extract the shared type into `domain` |
| `store` imports `controller` | upward | controller depends on `domain.Store`; provide impl in `cmd/` |
| `domain` imports `k8s.io/...` | core touches infra | move the K8s type into `store`/`runtime`; keep `domain` pure |
| package-level `var db = newPostgres()` | hidden wiring | construct in `cmd/` and inject via FX |

### Checking it

```bash
# fails if the import graph has a cycle
go list -deps ./... >/dev/null

# fails if domain imports anything outside the stdlib + own module
go list -f '{{.ImportPath}} {{join .Imports " "}}' ./domain/... | grep -E 'k8s\.io|net/http|database/sql' && exit 1 || true
```

Both run in CI. A red build here means a layer was crossed — fix the direction, don't add a
suppression.

> **ponytail:** the import-lint above is a grep, not a custom analyzer. If the allowlist
> grows unwieldy, swap in `go-arch-lint` or a `depguard` config — not before.
