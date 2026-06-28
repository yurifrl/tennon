# Bundles

A **bundle** composes a curated set of tools from one or more connectors behind a single endpoint,
optionally renaming tools and flagging which need approval. Bundles are how you give a team
exactly the tools they need without exposing whole connectors.

Backed by the [`Bundle`](crds.md#bundle) resource; built in the UI's drag-and-drop
composer (no YAML).

## Why bundles

- **Curate per use case.** A `devtools` bundle for engineers, a `support` bundle for the
  help desk — each with only the right tools.
- **Span multiple connectors.** Pull `search_*` from GitHub, `create_issue` from Jira, and a
  custom internal connector into one endpoint.
- **Rename for a clean surface.** Present consistent tool names to agents, decoupled from
  upstream naming.
- **One auth boundary.** Grant a team the bundle without granting the underlying connectors.

## How it resolves

When the controller reconciles a bundle, for each source it:

1. resolves `from` to a `Connector` and that connector's discovered tools,
2. applies `select` globs, intersected with the source connector's own allow/deny,
3. applies the `as` rename template to produce exposed names,
4. registers a route: exposed name → `(source connector, upstream tool)`,
5. publishes the bundle at `/bundle/<name>`.

```mermaid
graph LR
  B[Bundle: devtools]
  B -->|gh_search_*| GH[Connector: github]
  B -->|jira_create_issue| J[Connector: jira]
  B -->|rollback| D[Connector: internal-deploy]
```

## Approvals in a bundle

List tools under `approvals.require` and any call to them pauses for human approval before
forwarding — independent of the underlying connector's own settings. See
[Authorization → Approvals](authorization.md#approvals).

## Calling a bundle

```jsonc
{
  "mcpServers": {
    "mcphub-devtools": {
      "type": "streamable-http",
      "url": "https://mcphub.example.com/bundle/devtools",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```

The agent sees only the bundle's exposed tools. The proxy maps each call to the right
upstream connector, renders that connector's secret in-cluster, enforces policy, and pauses for approval
where required.
