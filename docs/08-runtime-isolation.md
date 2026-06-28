# Runtime & Isolation

How MCP servers are run, transported, and sandboxed. Two principles:

1. **Streamable HTTP everywhere, no SSE.** Every MCP is reached over the MCP **Streamable
   HTTP** transport. SSE is not used.
2. **Every MCP runs in a tightly jailed container** whose network and filesystem access is
   controlled by config. An MCP is treated as untrusted code.

## Transport: Streamable HTTP via Supergateway

Many MCP servers only speak **stdio**. To put them on the network as Streamable HTTP without
writing a bridge per server, MCPHub runs each stdio server behind
[**Supergateway**](https://github.com/supercorp-ai/supergateway) inside the same container.

- Supergateway wraps the stdio MCP process and exposes it as **Streamable HTTP** at `/mcp`:
  `supergateway --stdio "<server cmd>" --outputTransport streamableHttp --port 8000`.
- It ships as a container image (`ghcr.io/supercorp-ai/supergateway`, also `:uvx` / `:deno`
  variants), so it drops straight into the MCP pod/container as the entrypoint.
- MCPHub's proxy connects to that `/mcp` endpoint over Streamable HTTP. Servers that are
  already remote Streamable HTTP are used directly; **SSE is never used**.

So a container's shape is: `supergateway (Streamable HTTP :8000) → stdio MCP server`, and the
MCPHub proxy is the only client allowed to reach it.

## Where MCPs run (Docker locally, Pods in Kubernetes)

The isolation tools below apply in both environments; only the thing that launches the
container differs:

- **Local / single host:** MCP servers are **Docker containers** the controller starts and
  stops (gVisor runtime, `--read-only`, `--cap-drop ALL`, seccomp/AppArmor, an egress proxy
  or firewall rules). This is the solo / SQLite-Postgres mode — no cluster.
- **Kubernetes:** MCP servers are **Pods** created from the `McpConnector` CRD
  (`runtimeClass` for gVisor/Kata, `securityContext`, NetworkPolicy/Cilium, KubeArmor
  policy). The controller reconciles them like any operator.

The capability config you write is the same; the controller compiles it to Docker flags
locally or to Pod settings + policy CRDs in Kubernetes.

## Container isolation ("tightly jailed")

MCP code is untrusted, so containers are hardened in layers. All of this is config-driven per
`McpConnector`.

### Sandboxed runtime (kernel isolation)
Run MCP containers under a **sandbox runtime**, selected via a `runtimeClass`:

- **gVisor (`runsc`)** — intercepts syscalls in a user-space kernel; strong isolation with
  low overhead. Good default.
- **Kata Containers** — runs each container in a lightweight **microVM** (hardware
  virtualization, e.g. Firecracker/QEMU); strongest isolation, more overhead.

On a non-Kubernetes host, the equivalent is **Podman/Docker + gVisor**, or a process jail
like **nsjail** / **bubblewrap** for stdio servers.

### Pod/container hardening (always on)
- run as **non-root** (`runAsNonRoot`, a high UID),
- **read-only root filesystem**, writable only on explicit mounts,
- **drop ALL Linux capabilities**, `allowPrivilegeEscalation: false`, `no-new-privileges`,
- **seccomp** `RuntimeDefault` (or a tighter custom profile),
- **AppArmor**/**SELinux** profile,
- **resource limits** (CPU/memory/pids) via cgroups,
- no service-account token mounted; no host network/PID/IPC.

### Network egress control (what it can reach)
Default **deny all egress**, then allowlist per MCP. **We standardize on Cilium to start**,
with fallbacks where it isn't present:

- **Cilium (default)** — **FQDN/L7 egress policy**, e.g. allow only `grafana.example.com:443`.
  Profiles are written as domains, so an FQDN-aware layer is what enforces them faithfully.
- **Kubernetes NetworkPolicy** (fallback) — egress to resolved CIDRs/ports; coarser, no
  domain awareness.
- **Egress proxy** (Envoy/Squid) — domain allowlist off-cluster or where no CNI policy.

Ingress is locked down to just the MCPHub proxy. How a profile chooses among these (and what
the status reports) is the [Capability Enforcement design](capability-architecture.md).

### Filesystem access (what it can see/write)
- Read-only root; only declared mounts are present.
- Repo mounts attach a [Repo](repository-server.md design note) volume as `read` or
  `read-write`; that is the only way an MCP touches shared files. (See the Repository Server
  note, see [Repository Server](repository-server.md).)

## Config knobs (per MCP)

Surfaced on the `McpConnector` spec (set via the UI, no YAML required):

- `runtime.sandbox`: `gvisor | kata | none` (maps to a `runtimeClass`).
- `runtime.egress`: `deny` (default) + `allow: [host:port, CIDR]` (compiled to NetworkPolicy
  / Cilium FQDN / proxy allowlist).
- `runtime.filesystem`: read-only root + `mounts` (incl. Repo mounts with `read`/`read-write`).
- `runtime.resources`: CPU / memory / pids limits.
- `runtime.capabilities`: dropped by default; nothing added unless explicitly listed.

## What tools we use (summary)

- **Transport bridge:** Supergateway (stdio → Streamable HTTP, no SSE).
- **Sandbox runtime:** gVisor (default) or Kata Containers; nsjail/bubblewrap off-cluster.
- **Hardening:** Kubernetes `securityContext` (non-root, read-only rootfs, drop caps,
  no-new-privileges), seccomp, AppArmor/SELinux, cgroup limits.
- **Network control:** NetworkPolicy, or Cilium FQDN/L7 policy, or an egress proxy.
- **Filesystem control:** read-only rootfs + explicit (incl. Repo) mounts.
