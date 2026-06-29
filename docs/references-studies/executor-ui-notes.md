# Executor UI — scrape notes (localhost:4789, v1.5.13 BETA)

Live reference app for Tennon. Light theme, Instrument Serif page titles, Inter body,
JetBrains Mono for code. Token-gated (`auth.json` bearer; login screen: "paste the
server's token" or `executor open`).

## Layout
- Left sidebar (light, ~256px): `executor` wordmark + green **BETA** pill, stack icon.
  Nav: **Integrations · Secrets · Policies**. Section label `INTEGRATIONS` → connected
  providers list (Executor, `built-in`). Footer: Feedback/bug?, Star on GitHub, version.
- Main: serif H1 + muted subtitle, primary action top-right (green `+ Connect`).
- Cmd+K **command palette**: "Search integrations or jump to add…" → sections
  *Connected*, *Add integration* (Add OpenAPI / Add MCP / Add GraphQL), *Popular integrations*.

## Routes / screens
- `/` **Integrations** — "Tool providers available in this workspace."
  - "Connect an agent" card: **Remote HTTP** vs **Standard I/O** toggle.
    - Remote HTTP: `npx add-mcp http://localhost:4789/mcp --transport http --name executor --header 'Authorization: Bearer <token>'`
    - Standard I/O: `npx add-mcp 'executor mcp' --name executor` ("Requires the executor CLI on your PATH.")
    - **Advanced** expander → *Resume approvals*: "Standard I/O exposes a resume tool to the model. Use Remote HTTP for browser approvals." control = `Model resume tool`.
    - "Work with your agent [Claude/…/icons] and more".
  - Integration list rows: avatar + name + id (e.g. Executor / executor).
- **Connect modal** (`+ Connect`): "Search the preset library, or paste a URL to auto-detect."
  *Or add manually*: **OpenAPI · MCP · GraphQL**. *Popular integrations*: Google,
  Stripe, GitHub REST, Vercel — each tagged `OpenAPI`.
- `/secrets` **Providers** — "Where your connections' credential values live — the default
  store for pasted values, or an external backend like 1Password or your system keychain."
  - *Configure providers*: "Resolve secrets from your 1Password vault." → `Add 1Password`.
  - *Available providers*: Keychain (`keychain`), Local file (`file`), 1Password (`onepassword`) — each `provider` tag.
- `/policies` **Policies** — "Override default approval behavior for tools. The most
  restrictive matched action wins. Blocked tools are hidden from agent search and fail at invoke."
  - Add form: **Pattern** ("Exact tool id, trailing wildcard, or * for every tool.
    Examples: `*`, `vercel.*`, `vercel.dns.*`, `vercel.dns.create`") + **Action** select
    (**Require approval / Always run / Block**) + `Add policy`.
  - *Active policies*: "No policies yet. Tools fall back to their plugin's default approval behavior."
- `/integrations/:id` **integration detail** — header name + "N tools", **Tools** tab,
  master-detail: filter box ("Filter N tools…"), source group with count, right pane
  "Select a tool / Choose from the list to see what it does."
- `/tools` **Tools** — same master-detail, `Manage policies` button top-right.

## Functional takeaways for Tennon mock
1. **Secrets = pluggable credential providers** (default store / Keychain / Local file / 1Password) — distinct page, not just per-MCP inline secrets.
2. **Policies = pattern→action approval model** (Require approval / Always run / Block), most-restrictive-wins, blocked tools hidden from search and fail at invoke.
3. **Connect-an-agent onboarding**: copyable `npx add-mcp` for Remote HTTP vs stdio, with a resume-approvals advanced toggle.
4. **Integrations aren't only MCP**: OpenAPI / GraphQL / MCP, URL auto-detect + preset library.
