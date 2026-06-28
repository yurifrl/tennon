# Prior Art & Inspirations

MCPHub draws directly from two projects and sits alongside a broader set of MCP
gateway/security efforts.

## Primary inspirations

- [nsxbet/mcp-shield](https://github.com/nsxbet/mcp-shield) — the proxy + Kubernetes
  controller foundation, tool prefixing, and Kubernetes-native RBAC model. MCPHub keeps
  this skeleton but replaces the single config file with state-backed `Connector` resources,
  swaps the CLI-minted JWT for OAuth2 + service tokens.
- [RhysSullivan/executor](https://github.com/RhysSullivan/executor)
  ([DeepWiki](https://deepwiki.com/RhysSullivan/executor)) — the unified tool catalog,
  source auto-detection, intent-based discovery, and pause/resume (human-in-the-loop)
  execution model. MCPHub adapts these to a Kubernetes-native, multi-tenant control plane.

## Related work

- [MetaMCP](https://metamcp.com/) — unified middleware to manage multiple MCPs
  ([metatool-app](https://github.com/metatool-ai/metatool-app),
  [mcp-server-metamcp](https://github.com/metatool-ai/mcp-server-metamcp)).
- [Smithery](https://smithery.ai/docs) — MCP registry/hosting.
- [nshkrdotcom/GUARDRAIL](https://github.com/nshkrdotcom/GUARDRAIL) — gateway for unified
  access, resource delegation, and risk limits.
- [eqtylab/mcp-guardian](https://github.com/eqtylab/mcp-guardian) — manage / proxy /
  secure MCP servers.
- [Teleport MCP access RFD](https://github.com/gravitational/teleport/blob/master/rfd/0209-mcp-access.md)
  — access-control design for MCP.

## How MCPHub differs

| Dimension | MCP Shield | Executor | MCPHub |
|-----------|-----------|----------|--------|
| Config source | Single YAML file | Local store | State-backed `Connector` CRDs |
| Auth | CLI JWT via SSO | Shared local creds/OAuth | OAuth2 + service tokens |
| Authorization | `MCPPermission` CRDs | Policies | `Policy` CRDs |
| Aggregation | One proxy endpoint | One catalog | Per-MCP URLs |
| Secrets | Env from config | Keychain | `Connector` `secretKeyRef`, rendered in-cluster |
| Approvals | — | Pause/resume | Pause/resume on sensitive tools |
| UI | Roadmap | React dashboard | Vite/React (embedded in `mcphub`) |
| Runtime | K8s containers | QuickJS sandbox | K8s containers + remote MCPs |
