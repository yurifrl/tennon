# CLAUDE.md — Agent directives

Read `docs/` for the full design. This file is the short list of rules an agent must not
break. When in doubt, the docs win; when the docs are silent, these directives do.

## Prime directive: decoupling

Every feature is a new file that implements an existing seam, not a change that reaches
across one. If your change makes package A know about package B that it didn't before, stop —
you're coupling. Extract the shared thing into `domain` and depend on the interface.

- **Packages only call down** (see `docs/arch/03-project-structure.md`). `domain` at the bottom
  imports nothing outward. Infra (`store`, `runtime`, `secret`, `auth`) implements `domain`
  ports. Delivery (`api`, `proxy`, `controller`, `cli`) calls down into ports, never sideways.
- **Control flows up only through interfaces the lower layer owns.** The proxy needs an
  authz decision → it depends on `domain.Authorizer`, not on the auth package.
- **One composition root.** Concrete types are wired in `cmd/` via Uber FX. No
  package-level `var`, no `init()` wiring, no constructing your own dependencies.

A change that crosses a layer is a bug even if it compiles. CI fails on import cycles and on
`domain` importing infra — don't add a suppression, fix the direction.

## The core engine must run without auth or authz

This is non-negotiable and the reason the seams exist. The MCP engine — proxy + container
runtime + connectors + store — must boot and serve MCP with **authentication off and
authorization off**. In that mode:

- The `IdentityVerifier` is a no-op that resolves every caller to a fixed anonymous
  `domain.Subject`.
- The `Authorizer` is allow-all (`Decide` → `allow`).
- No OIDC, no tokens, no Cedar, no approvals queue. Just MCP-over-HTTP routing to containers.

Wire this via FX in `cmd/`: a flag (e.g. `--auth=off`, `--authz=off`) selects the no-op
implementations. The core packages must not import the auth/authz packages — they only see
the `domain` ports. If turning auth off requires touching `proxy` or `runtime`, the seam is
wrong.

Default-deny (`docs/09-authorization.md`) is a property of the *real* `Authorizer`, not of
the proxy. The proxy asks; the injected authorizer answers. Swap the authorizer, swap the
behavior.

## Auth and permissions are plugins, not features

The user/identity and policy systems are pluggable implementations behind these `domain`
ports (`domain/ports.go`):

- `IdentityVerifier` — turns a presented credential into a `Subject` (OIDC/Okta, K8s SA,
  SPIFFE, token minting, or the no-op).
- `Authorizer` — turns `(Subject, ToolCall)` into an `allow | ask | block` `Decision`
  (Cedar-compiled policy, or allow-all).

Adding a new identity provider or a new policy engine = a new implementation of one port,
provided in `cmd/`. It is never an edit to `proxy`, `runtime`, `store`, or `domain`.

## Two teams, one contract

Core engine and auth/authz are built in parallel. The **only** shared surface is the
`domain` port interfaces and types (`Subject`, `ToolCall`, `Decision`, `IdentityVerifier`,
`Authorizer`). Agree on those signatures first, then both sides build against them
independently — the engine against the no-op impls, the auth team against the real ones.

Rules that prevent a coding clash:

- Change a `domain` port signature → it's a contract change. Surface it explicitly; both
  sides re-agree. Never silently widen an interface to make your side compile.
- The engine never imports the auth packages. The auth packages never import `proxy`/
  `controller`/`api`. They meet only at `domain` and get composed in `cmd/`.
- New behavior that the engine needs from "above" → add a port the engine owns in `domain`,
  let the other side implement it. Don't reach up.

## Hooks for extension

When the engine needs a decision or a side effect it shouldn't own (authz, audit, secret
rendering, approval routing), it calls a `domain` port — that *is* the hook. Add a hook the
same way: define the interface in `domain` next to its callers, default it to a no-op impl,
wire the real one in `cmd/`. A hook with one real implementation still earns its place
*only* because the no-op keeps the core runnable without it; don't add hooks for hypothetical
extension points (YAGNI).

## Docs: `docs/` is the user guide, `docs/arch/` is the design

`docs/*.md` (root, served at `/`) is for someone installing and using MCPHub: how to install,
the concepts they touch, and how to use each feature (UI, CLI, API, connectors, auth, policies,
tokens, CRDs reference, storage, deployment). No architecture or rationale.

`docs/arch/*.md` (served at `/arch`) is for someone building MCPHub: requirements, architecture,
project structure, technologies, internals (proxy, runtime isolation, capability/CRD design),
roadmap, contributing, and research notes. Design decisions a user doesn't need go here.

Adding a doc: pick the bucket, drop in an `NN-name.md`, run `task docs:build` (regenerates both
`_all.md` pages). Cross-doc links use the un-numbered name (`](cli.md)`); the build rewrites them.

## Checklist before you call it done

- [ ] Did I add an import that points up or sideways? (no)
- [ ] Does the core still boot with `--auth=off --authz=off`? (yes)
- [ ] Is every concrete type I introduced wired in `cmd/`, not constructed inline? (yes)
- [ ] If I touched a `domain` port, did I flag it as a contract change? (yes)
- [ ] `go list -deps ./...` acyclic and `domain` import-lint clean? (yes)
