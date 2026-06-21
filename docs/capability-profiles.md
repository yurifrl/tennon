# Sandbox & Access (Capability Profiles)

Every MCP runs in a locked-down container. A **Capability Profile** is how you say, in plain
terms, **what that container is allowed to do** — what it can reach on the network, which
files it can touch, and what it can run. Anything you don't list is denied.

You write *intent*. MCPHub enforces it with whatever the cluster has (today: Cilium for
network, plus process/file and sandbox enforcement); you never name those tools yourself.

## The mental model

> List only what it's **allowed** to do. Everything else is blocked automatically.

You never write "block X." You list the few things the MCP legitimately needs, and that's the
whole allowlist.

## What you set

- **Sandbox** — how hard the container is isolated: `Strong` (default), `Maximal`, or `Off`.
- **Can reach** — network destinations it may call out to, e.g. `grafana.example.com:443`.
- **Can touch** — files/paths and Repo mounts it may read or write.
- **Can run** — which programs may execute (defaults to just the MCP server).
- **Enforcement** — `Require` (refuse to start if it can't be enforced) or `Best effort`.

## Writing one

Three equivalent ways — pick what fits.

### In the web UI

On an MCP, open the **Sandbox & access** panel and fill it in:

```
Sandbox:      [ Strong ▾ ]
Can reach:    + grafana.example.com:443         (everything else blocked)
Can touch:    + /etc/grafana          [read]
              + repo: notes → /vault  [read-write]
Can run:      (default: the server)   + /app/grafana-mcp
Enforcement:  (•) Require   ( ) Best effort
Status:       ✓ Fully enforced
```

A dropdown, a few rows you type into, one radio. No syntax.

### From the CLI

```bash
mcphub profile create grafana-jail \
  --sandbox strong \
  --allow-net  grafana.example.com:443 \
  --allow-file /etc/grafana:read \
  --allow-repo notes:/vault:rw \
  --allow-run  /app/grafana-mcp \
  --enforce require
```

### As a text file (GitOps)

```yaml
profile: grafana-jail
sandbox: strong
allow:
  network: [ grafana.example.com:443 ]
  files:
    - /etc/grafana          read
    - repo:notes -> /vault  read-write
  run: [ /app/grafana-mcp ]
enforce: require
```

All three produce the same profile object.

## Attaching it to an MCP

- **UI:** on the MCP, set **Capability profile: `grafana-jail`** (a dropdown).
- **CLI:** `mcphub mcp set grafana --profile grafana-jail`.

One profile can be reused across many MCPs, or you can edit it inline on a single MCP. If you
don't attach one, a safe default profile is applied.

## Enforcement: Require vs Best effort

Clusters differ in what they can enforce. You choose how strict to be:

- **Require** — if any part of the profile can't be truly enforced here, the MCP **refuses to
  start**. Use this for anything credentialed or sensitive.
- **Best effort** — the MCP runs, and the **Status** line tells you exactly what is fully
  enforced and what is only audited or weaker on this cluster. No silent gaps.

## The status line

The one place the cluster's reality shows up, in plain language:

- `✓ Fully enforced` — everything in the profile is actively blocked where it should be.
- `⚠ Network limited to IPs` — egress is allowed, but by address rather than domain (a
  network add-on isn't present).
- `⚠ Process rules audited only` — exec rules are logged but not blocked on this cluster.

You read intent + status; MCPHub handles the rest. For how that translation works, see
[Capability Enforcement (design)](capability-architecture.md).
