# Executor — Complete Feature Specification

A feature inventory of [RhysSullivan/executor](https://github.com/RhysSullivan/executor),
derived from the cloned source at `.references/executor` (README, vision.md, AGENTS.md,
RUNNING.md, the published docs under `apps/docs`, and the actual package sources). This is
a reference catalog of *every* feature, intended as input for designing a similar system.

> **What it is.** Executor is "the integration layer for AI agents: one catalog for every
> tool, shared across every agent you use." It is a local-first (also cloud-hostable)
> control plane that turns any API/MCP/GraphQL source into a unified, typed, authenticated,
> policy-governed tool catalog that agents call through code or MCP. It is explicitly *not*
> AI-specific and *not* code-mode-specific (vision.md): it is a category of "source" and a
> way to interop between them.

---

## 1. Core Concepts

Executor's data model (verified in `packages/core/sdk/src`). The user-facing docs
(`apps/docs/concepts`) name three; the SDK adds the underlying primitives.

| Concept | Definition | Key fields / behavior |
|---------|-----------|------------------------|
| **Integration** | Something Executor connects to, defined by a *source* (MCP server, OpenAPI spec, or GraphQL endpoint). Describes the tool catalog; not itself a live connection. | `slug`, `name`, `description`, `kind` (plugin type), opaque plugin `config`, derived `authMethods`, `displayUrl`, `canRemove`, `canRefresh`. (`integration.ts`) |
| **Connection** | A configured instance of an integration. One integration can have many connections (e.g. same API, two accounts). Unifies secret + account + connection into one row. | `owner`, `name`, `integration`, `template` (auth method), `provider` (credential backend), `description`, `identityLabel`, `expiresAt`. Address: `tools.<integration>.<owner>.<connection>.<tool>`. Never stores the secret value itself, only resolver routing. (`connection.ts`) |
| **Tool** | A persisted, addressable callable owned by a connection (or static). | `address`, `owner`, `integration`, `connection`, `name`, `inputSchema`, `outputSchema` (JSON Schema), annotations `requiresApproval`, `approvalDescription`, `mayElicit`. (`tool.ts`) |
| **Secret** | Credential material. In v2 it is folded into Connection; the actual bytes live in a *credential provider* backend, never in core. | Resolved via `connections.resolveValue` through the connection's `provider`. |
| **Source** | The thing that contains tools (OpenAPI/GraphQL/MCP/Google Discovery, or any plugin type). | Backed by protocol plugins; "if you can represent it with a JSON schema, it can be an integration." |
| **Scope / OwnerBinding** | Multi-tenant scoping. v2 replaces scope-stacking with a single `OwnerBinding` = `tenant` + `subject` (nullable). | Plugin storage, blob store, and policy resolution are all scoped to this binding. |
| **Policy** | Per-tool governance: allow / require approval / block. | See §5. |
| **Plugin** | Registers tools, sources, adapters, credential providers, auth methods. | See §6. |
| **Manager / Invoker / Executor** | The runtime that lists tools, executes, and routes through plugins. | `createExecutor(plugins)` → typed instance; `tools.list()`, `execute()`, `integrations.*`, `connections.*`, `policies.*`. (`executor.ts`) |

---

## 2. Tool Discovery & Invocation

- **Unified tool catalog.** Every tool from every source is indexed into one searchable,
  addressable catalog, shared across all connected agents.
- **Intent-based discovery.** `tools.discover({ query, limit })` finds tools by natural
  language; `executor tools search "<query>"` from the CLI.
- **Schema inspection.** `tools.describe.tool({ path, includeSchemas })` /
  `executor tools describe <path>` returns the tool's TypeScript/JSON schema.
- **Typed runtime calls.** Agents call tools as a typed TS namespace tree:
  `tools.github.issues.list({ owner, repo })` with type safety.
- **CLI invocation by path.** `executor call <path...> '{json args}'`; `--help` browses
  namespaces/resources/methods; `--match <text> --limit <n>` narrows huge namespaces.
- **Source listing.** `executor tools sources` lists configured integrations + tool counts.
- **Auto-detection on add.** Paste a URL (web UI **Add Source** or
  `executor call executor openapi addSource {...}`); Executor detects the type, indexes
  tools, and handles auth. `baseUrl` handles relative OpenAPI `servers` entries.

---

## 3. Execution Model: Pause / Resume / Approval (Human-in-the-loop)

Verified in `packages/core/execution` and `packages/core/sdk/src/elicitation.ts`.

- **Execution result** is either `{ status: "completed", result }` or
  `{ status: "paused", execution }`.
- **Elicitation** suspends the running effect *fiber* (not the HTTP request) when a tool
  needs the user. Two kinds:
  - **UrlElicitation** — send the user to a URL (OAuth consent, an approval page); resume
    on callback.
  - **FormElicitation** — request structured input matching a `requestedSchema` (JSON Schema).
- **Resume actions:** `accept` (with optional `content`), `decline`, `cancel`.
- **PausedExecution** carries an `id`, the `elicitationContext` (address, args, request),
  and holds the fiber + a `Deferred` that the resume completes.
- **CLI resume:** `executor resume --execution-id <id> [--action accept|decline|cancel] [--content ...]`.
- **Pauses happen for both auth and approval** — the same mechanism covers "needs login"
  and "needs human sign-off."

---

## 4. Authentication & Credentials

- **HTTP auth vocabulary** (`packages/core/sdk/src/http-auth`) is *placements-based*:
  - Carrier: `header` or `query`.
  - Target: header/param `name`, optional `prefix` (e.g. `"Bearer "`).
  - Binding: a credential `variable` (default `"token"`) or a static `literal`.
  - `renderAuthPlacements()` produces `{ headers, queryParams }`.
- **Auth method kinds:** `apikey` (ApiKeyAuthMethod with placements), `none`
  (NoneAuthMethod), plus OAuth modeled per-plugin.
- **Connections store credentials once** and apply them on every call, so the agent never
  handles secrets directly. Public APIs/MCP servers can connect with no credentials.
- **Pluggable credential providers** (secret backends) — see §6.
- **Shared OAuth service** exposed to plugins via `PluginCtx.oauth`; OAuth connections
  carry `expiresAt`.

---

## 5. Policies (per-tool governance)

Verified in `packages/core/sdk/src/policies.ts` and `apps/docs/concepts/policies.mdx`.

- **Three actions per tool:** **Allow** (runs uninterrupted), **Require approval** (pauses
  for a human), **Block** (cannot be called).
- **Sensible defaults from the spec:** e.g. read-only `GET` OpenAPI operations default to
  allow; writes can default to require-approval.
- **Rule fields:** `id`, `owner`, `pattern`, `action` (`approve` | `require_approval` |
  `block`), `position` (fractional-index precedence).
- **Pattern matching:** `*` (universal), exact segments, `vercel.*` (plugin-wide), trailing
  `*` (subtree).
- **Owner-scoped layering:** per-owner first match by position; **most restrictive action
  across owners wins** (org guardrail over user preference). `resolveEffectivePolicy()`
  layers user rules over the plugin's default `requiresApproval`.

---

## 6. Plugin System

`packages/core/sdk/src/plugin.ts`. A plugin can register tools, sources, adapters, auth
methods, and credential providers.

- **Lifecycle hooks:** `storage()` (typed per-plugin store factory), `extension()`
  (extension API surfaced as `executor[pluginId]`).
- **Static tools:** `staticSources()` contribute inline handlers (always present).
- **Dynamic tools:** `resolveTools()` (connection → `ToolDef[]`), `invokeTool()` (execute).
- **Auth:** `credentialProviders` (storage backends), `describeAuthMethods()` (project
  config into catalog auth methods).
- **Plugin context (`PluginCtx`)** provides: `owner`, `storage`, `pluginStorage`,
  `httpClientLayer`, and core namespaces — `integrations` (CRUD), `policies` (CRUD),
  `connections` (create/list/get/update/remove/refresh/`resolveValue`), `providers`
  (list backends + items), `oauth`, and `elicit` (suspend/resume for user interaction).
- **Transactional storage:** `transaction()` spans plugin storage + core writes.

### 6.1 Protocol / source plugins
- **openapi** — introspect & invoke OpenAPI/REST specs (YAML/JSON), with OAuth2 / API key
  / Bearer auth.
- **graphql** — introspect & invoke GraphQL servers (introspection query → fields/mutations),
  variables + auth.
- **mcp** — discover & invoke MCP servers; stdio and remote transports; env-var or bearer auth.
- (First-party support also advertised for **Google Discovery**.)

### 6.2 Credential-provider (secret store) plugins
- **encrypted-secrets** — AES-256-GCM in the executor `plugin_storage` table; scrypt key
  derivation from a master secret; multi-tenant safe (used by self-host/cloudflare hosts).
- **file-secrets** — plaintext JSON credential file (`~/.local/share/executor/auth.json`,
  or `%LOCALAPPDATA%` on Windows); simple single-user local/desktop store.
- **keychain** — OS keychain (macOS Keychain, Linux Secret Service, Windows Credential
  Manager) via `@napi-rs/keyring`.
- **onepassword** — 1Password via `@1password/sdk` or the `op` CLI / service account.
- **workos-vault** — WorkOS Vault API via `@workos-inc/node`.

### 6.3 Other plugins
- **desktop-settings** — Electron browser-handoff tool routing to the desktop settings UI.
- **example** — minimal reference plugin (extension methods, HttpApi groups, late-binding
  handlers).

---

## 7. Kernel — Code Execution Substrate ("code mode")

`packages/kernel/*`. Lets an agent write arbitrary code that calls `tools.*`, run in a
sandbox instead of having raw shell access.

- **kernel/core** — the code-mode contract: shared `Tool`, `SandboxToolInvoker`,
  `CodeExecutor` types + JSON Schema helpers. New runtimes implement `CodeExecutor`.
- **kernel/ir** — intermediate representation for tool catalogs (serialize/deserialize tool
  metadata, schema extraction, registry queries).
- **Three runtime backends:**
  1. **runtime-quickjs** — WASM QuickJS interpreter; isolated VM with configurable timeout,
     memory, and stack limits; the default for untrusted LLM-generated code.
  2. **runtime-deno-subprocess** — spawns a Deno subprocess, IPC over an `@@executor-ipc@@`
     prefix, Deno permission flags (e.g. `--allow-net`), ~5-min default timeout.
  3. **runtime-dynamic-worker** — Cloudflare Worker isolation; dispatches tool calls over
     Workers RPC (`ToolDispatcher` RpcTarget); network isolation via `globalOutbound`;
     used by the cloud deployment.

---

## 8. MCP Host — Exposing Executor as an MCP Server

`packages/hosts/mcp`. `executor mcp` serves a streamable-HTTP MCP endpoint (and stdio via
the CLI) so any MCP client (Claude Code, Cursor, OpenCode) shares one catalog, auth, and
policy set.

- **Exposed tools:**
  - `execute(code)` — run TS/JS that calls `tools.*`; returns results or pauses on
    elicitation/approval.
  - `resume(executionId, action, content?)` — resume a paused execution.
  - (plus the discovery/describe surface from §2)
- **Elicitation modes** (`?elicitation_mode=`): `model` (agent resumes inline), `browser`
  (agent polls; user approves in a browser UI at `/resume/:executionId`, 10-min timeout),
  `native` (integration handles elicitation directly; no resume tool).
- **BrowserApprovalStore** — paused-execution state: `takeResponse`/`waitForResponse`.
- **In-memory session store** — tracks paused executions, expires stale ones, rebuilds on
  host restart.
- **Add-to-client helper:** `npx add-mcp http://127.0.0.1:17888/mcp --transport http --name executor`,
  or `npx add-mcp "executor mcp"` for stdio.
- **Pi bridge:** Pi has no built-in MCP client, so a community bridge
  (`pi-executor-mcp`) connects Pi to Executor.

---

## 9. CLI Surface

The CLI runs a small local HTTP runtime (durable background service). Commands
(`apps/cli/src`, README, `apps/docs/local/cli.mdx`):

| Command | Purpose |
|---------|---------|
| `executor install` | Install/start the durable background service (survives restarts) |
| `executor web` | Open the running web UI (default `http://127.0.0.1:17888`) |
| `executor web --foreground` | Throwaway foreground runtime + web UI |
| `executor daemon run` | Start persistent local daemon in background |
| `executor daemon status` / `stop` / `restart` | Manage the daemon |
| `executor mcp` | Start the MCP endpoint |
| `executor call <path...> '{json}'` | Invoke a tool by path segments |
| `executor call <path...> --help [--match <t> --limit <n>]` | Browse / narrow namespaces |
| `executor resume --execution-id <id>` | Resume a paused execution |
| `executor tools search "<query>"` | Search tools by intent |
| `executor tools sources` | List sources + tool counts |
| `executor tools describe <path>` | Show a tool's TS/JSON schema |

Behavioral details: `call`/`resume`/`tools` **auto-start the daemon** if needed; if the
default port is busy the CLI **picks and tracks a free port**; daemon startup is serialized
behind a filesystem lock to prevent duplicates; the web UI is **embedded into the compiled
binary**.

---

## 10. Web UI & SDK

- **Web UI** (`packages/app`, `packages/react`, served by local/host apps) — Add Source,
  catalog browser, connection/credential management, policy tuning, and browser approval
  inbox. React + atom/client integration; built with a custom `@executor-js/vite-plugin`.
- **SDKs / programmatic API:**
  - `@executor-js/sdk` core (exports `.`, `./core`, `./client`, `./promise`, `./testing`,
    `./migration`, `./http-auth`, `./public-origin`, `./host-internal`).
  - `@executor-js/api` (`./client`, `./server`).
  - `@executor-js/execution` (`.`, `./promise`).
  - Examples: `examples/promise-sdk`, `examples/docs-sdk-quickstart`, `examples/all-plugins`.

---

## 11. Integrations Registry

`packages/core/integrations-registry`. A curated, cached catalog of known integrations.

- Service: `get()` (cached data) / `refresh()`.
- Config: `EXECUTOR_INTEGRATIONS_URL` override, cache dir, 12h TTL, disable flag.
- Cache: JSON per source URL (default `~/.executor/cache/integrations.json`), disk-based
  with a lock file.
- Stable User-Agent: `executor/<channel>/<version>/<client>` (stable/beta/dev; cli/local/desktop).

---

## 12. Deployment Targets (Apps)

| App | What it is |
|-----|-----------|
| `apps/local` | Local dev/desktop layer: Vite+React UI + in-process Executor server, QuickJS runtime, file-based secrets. |
| `apps/desktop` | Electron app wrapping the local server as a sidecar; system keychain + 1Password; Sentry crash reporting. |
| `apps/host-selfhost` | Self-hosted Bun HTTP server; Vite+React; encrypted-secrets; better-auth API-key auth; LibSQL/SQLite. |
| `apps/host-cloudflare` | Cloudflare Pages + Worker; minimal bundle; QuickJS; encrypted-secrets. |
| `apps/cloud` | Multi-tenant SaaS on Cloudflare (Workers + Durable Objects); PostgreSQL; dynamic-worker runtime; WorkOS auth; OpenTelemetry. |
| `apps/cli` | The npm-distributed CLI / daemon (embeds the web UI). |
| `apps/marketing`, `apps/docs` | Marketing site and Mintlify docs. |

---

## 13. Billing / Plans (cloud)

`autumn.config.ts` (Autumn metering). Metered feature **executions** (consumable); boolean
features like **domain-verification** and **members**. Plans: **free**, **freePayAsYouGo**,
**team**, **enterprise**.

---

## 14. Testing, Emulators & Security Tooling

- **Service emulators** — `@executor-js/emulate` (separate repo
  `UsefulSoftwareCo/emulate`): wire-level, stateful emulators for GitHub, Google, Stripe,
  Resend, WorkOS, and more. Each serves a full OpenAPI spec, mints real-shaped credentials,
  runs working OAuth flows, and records a request ledger to assert against. Hosted at
  `https://<service>.emulators.dev`.
- **mcporter** — `@executor-js/mcporter` (separate repo), a headless MCP client used in tests.
- **Test stack:** Effect Vitest (unit/integration), Playwright/Chromium (e2e). E2e boots
  both cloud and self-host apps and drives them; runs produce screenshots, `session.mp4`,
  and `trace.zip`; a viewer serves the scenario × target matrix.
- **Dev CLI** (`e2e bun run cli`) — boot a target, mint identities, make typed API calls,
  drive MCP, read the emulator ledger, share over tailnet.
- **Warden** (`warden.toml`) — security review tooling (`failOn = high`, `reportOn =
  medium`) with a `warden-security-review` skill.
- **Tooling:** Bun workspaces + Turbo, Changesets (releases), oxlint/oxfmt, Prettier, Knip.

---

## 15. Planned / Vision Features (vision.md)

Not necessarily shipped — the stated direction:

- **Dynamic plugin support** — run plugins in a V8 isolate; let the agent write whatever it
  needs; ship instructions as part of executor (see `notes/dynamic-plugin-loading-v1.md`).
- **Integrations registry** (partially shipped, §11).
- **Configure executor via executor** (manage Executor through its own tools).
- **MCP apps with dynamic UI** — React Flight / RSCs + code mode (see
  `notes/artifacts-workflows-and-generated-ui.md`, `notes/mcp-apps-executor-gateway.md`).
- **Scope merging** — add tools at global/workspace/account level, override secrets and
  policies per scope, create temporary scopes.
- **Workflows** — ideally built on a "use workflow" primitive.
- **SDK** (shipping, §10), **internal apps catalog**, **custom UI snippets store**.
- **MCP channels** — talk back to the agent mid-tool-call (like Claude Code over Discord).
- **Per-chat storage** — temporary KV / SQLite / filesystem the agent can use and persist
  on scopes between tool calls.

> Guiding principle (vision.md): "ship the primitives that build an extendable product."

---

## 16. Feature Summary (one-line index)

1. Unified, addressable tool catalog across all sources and agents.
2. Integration → Connection → Tool data model (multi-account, credential-isolated).
3. Source auto-detection + indexing (OpenAPI, GraphQL, MCP, Google Discovery; plugin-open).
4. Intent-based discovery, schema inspection, typed namespace calls.
5. Code-mode execution in sandboxed kernels (QuickJS / Deno subprocess / CF worker).
6. Pause/resume elicitation (URL + form), for both auth and human approval.
7. Per-tool policies (allow / require-approval / block) with owner-layered, most-restrictive-wins.
8. Placements-based HTTP auth + pluggable credential providers (keychain, 1Password, WorkOS, encrypted, file).
9. Plugin system registering tools/sources/auth/providers, with scoped transactional storage.
10. MCP host: expose Executor itself as an MCP server with `execute`/`resume` + 3 elicitation modes.
11. CLI + durable daemon (auto-start, free-port tracking, embedded web UI).
12. Web UI (add source, browse, manage credentials, tune policies, approve in browser).
13. Multiple deployment targets: local, desktop (Electron), self-host (Bun/SQLite), Cloudflare, multi-tenant cloud.
14. Integrations registry (cached, refreshable, configurable).
15. Billing/metering (Autumn) with free/team/enterprise plans.
16. Wire-level service emulators + headless MCP client + e2e harness + Warden security review.
