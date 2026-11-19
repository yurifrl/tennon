# Tennon

The Kubernetes-native control plane and proxy for Model Context Protocol (MCP) servers.

Register MCP servers, share them safely (credentials never leave the platform), and govern
every tool call — driven from a web UI, CLI, or API that all write the same objects.

Full documentation lives in [`docs/`](docs/00-README.md); architecture and internals are under
[`docs/arch/`](docs/arch/README.md).

## Status

Phase 1 of the [roadmap](docs/arch/10-roadmap.md) is implemented: **register a connector,
reconcile it, and proxy its tools at `/mcp/<name>` using the connector's rendered credential —
never exposing that credential to the caller.** Later phases (spec connectors, pluggable
storage/auth, Cedar authorization, tokenless identity, approvals, and the React UI) are tracked
in Beads (`bd ready`).

## Quickstart

```bash
task build                       # -> bin/tennon, bin/tctl

# Run a local mock MCP upstream (for trying the flow):
go run ./hack/mockmcp --addr 127.0.0.1:9090

# Run the control plane (solo / local, in-memory store):
./bin/tennon --store memory --runtime remote \
  --api-addr 127.0.0.1:8080 --proxy-addr 127.0.0.1:8081

# Register a credentialed connector and inspect it:
export TENNON_API=http://127.0.0.1:8080
./bin/tctl secret create grafana-creds --from-literal token=$GRAFANA_TOKEN
./bin/tctl connector add grafana \
  --remote http://127.0.0.1:9090/mcp \
  --header "Authorization=Bearer {{secret:grafana-creds.token}}" \
  --expose-prefix grafana_ --allow "get_*" --tag observability
./bin/tctl connector list        # -> grafana  remote  Ready  /mcp/grafana
```

Agents then point an MCP client at `http://127.0.0.1:8081/mcp/grafana`; the proxy injects the
connector's credential server-side, so the caller never holds it.

For persistence use `--store sqlite --store-path tennon.db`.

## Development

```bash
task ci      # layering check + go vet + tests
task test    # tests only
task check   # layering rules only
```
