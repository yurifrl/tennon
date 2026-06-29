# Connector Extensibility (design note)

> Design decision for how Tennon supports new and provider-specific connectors. Captures the
> options considered and the chosen path. See [Spec Connectors](spec-connectors.md) for the
> mechanics this builds on.

## The question

How do we support "special" connectors — Google, AWS, a provider with an odd auth or
pagination shape — and new source types over time? The tempting answers are a plugin system,
WASM, an embedded sublanguage, or runtime-loaded code. Most of them are the wrong tool, and
the reason is that **"special" is mostly not code.**

## Decompose "special" first

Looking at what real providers actually need (Google being the worked example), the variation
splits into four needs, and only one of them wants arbitrary code:

| Need | Example | Right mechanism |
|------|---------|-----------------|
| Auth shape | Google OAuth2 scopes/audience | **config / data** (`auth` block + `OAuthApp`) |
| Curated catalog & UX | the Google service list, icons, scopes | **data** (presets) |
| Field / argument / error / pagination mapping | rename a field, page results, map errors | **embedded expression language** |
| New spec format or wire protocol | Google Discovery → OpenAPI, gRPC reflection | **code (a built-in adapter)** |

The first two are data. The third is bounded expression evaluation. Only the fourth is code,
and its cardinality is low (a handful of spec formats over the life of the project).

## Options considered

- **Hardcode — built-in adapters behind an interface.** A small Go interface
  (`SpecAdapter: source → tools`, an implementation of the connector's `ToolProducer` seam)
  with trusted in-tree implementations (OpenAPI, GraphQL, Google Discovery, later gRPC). A new
  format is a PR + release. Type-safe, no sandbox, no attack surface. This is effectively what
  the reference system (executor) ships. **Chosen baseline.**
- **Embedded expression language (CEL).** For the config-level transforms — credential
  templating, argument/field mapping, `when` conditions, pagination rules. We already use this
  shape (`{{secret:…}}`, Cedar `when`). CEL is non-Turing-complete, so it is safe to evaluate
  without a sandbox. (Starlark is the fallback if real procedural logic is ever needed.)
  **Chosen for transforms.** It cannot implement a new wire protocol — that stays code.
- **WASM.** Genuinely sandboxed, hot-loadable, language-agnostic — the right tool *only* if we
  expect many **untrusted third-party** connector binaries. For converting a Discovery
  document it is large overkill (component-model interface, debugging, immature Go host
  story). **Deferred.**
- **Out-of-process plugins (gRPC, HashiCorp go-plugin / Terraform-provider style).** Robust
  and isolated, but in Kubernetes each plugin becomes another container/sidecar with lifecycle
  and distribution cost. Justified only when a third-party provider ecosystem must extend the
  binary without forking it. **Deferred.**
- **Go `plugin` package / runtime-loaded ("JIT") code.** Fragile (version-locked, no unload)
  and pure attack surface for no benefit here. **Rejected.**

## Decision

For v1, do **not** build a general plugin/WASM/JIT system. Instead:

1. **Push "special" into data** — a presets/catalog surface (service lists, scopes, icons)
   that needs zero code, the way Google is handled.
2. **Hardcode the few spec adapters** as Go implementations behind the connector's
   `ToolProducer`/`SpecAdapter` interface. Google is an `OpenApiConnector` + the Discovery
   adapter + a preset — **not** a new connector kind.
3. **Add one embedded expression language (CEL)** for credential/field/condition transforms,
   so the config layer is expressive without code.

Because connectors are composed from interfaces (`ToolProducer`, `Invoker`, `Runtime`,
`Credentials`), a WASM or gRPC-plugin backend is an **additive future** — a new implementation
of one interface, not a remodel. We keep that seam clean now and ship hardcoded adapters.

## Revisit triggers

Reconsider WASM or out-of-process plugins only when one of these is true:

- Third parties need to ship connector logic **without forking** the controller binary.
- Users need to run **untrusted** transform/adapter code (sandboxing becomes a requirement,
  not a nicety).
- The set of spec adapters grows past what is comfortable to maintain in-tree.

Until then: hardcode + data + CEL, and keep the interface seam. YAGNI.
