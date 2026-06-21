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

## Container isolation ("tightly jailed")

MCP code is untrusted, so containers are hardened in layers. All of this is config-driven per
`Mcp` / `McpConnection`.

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
Default **deny all egress**, then allowlist per MCP. Options, increasing in power:

- **Kubernetes NetworkPolicy** — allow egress only to specific CIDRs/namespaces/ports
  (e.g. the upstream API and DNS); deny everything else, including reaching other MCPs.
- **Cilium** (or other L7 CNI) — **FQDN/L7 egress policy**, e.g. allow only
  `api.github.com:443`; domain-level allowlists map cleanly onto config.
- **Egress proxy** (Envoy/Squid) — force all traffic through a proxy that enforces a domain
  allowlist; works on non-Kubernetes hosts too.

Ingress is locked down to just the MCPHub proxy.

### Filesystem access (what it can see/write)
- Read-only root; only declared mounts are present.
- Repo mounts attach a [Repo](repository-server.md design note) volume as `read` or
  `read-write`; that is the only way an MCP touches shared files. (See the Repository Server
  note in `specs/`.)

## Config knobs (per MCP)

Surfaced on the `Mcp` / `McpConnection` spec (set via the UI, no YAML required):

- `runtime.sandbox`: `gvisor | kata | none` (maps to a `runtimeClass`).
- `runtime.egress`: `deny` (default) + `allow: [host:port, CIDR]` (compiled to NetworkPolicy
  / Cilium FQDN / proxy allowlist).
- `runtime.filesystem`: read-only root + `mounts` (incl. Repo mounts with `read`/`read-write`).
- `runtime.resources`: CPU / memory / pids limits.
- `runtime.capabilities`: dropped by default; nothing added unless explicitly listed.

## Capability policy & enforcement (KubeArmor / ModelArmor)

The human-authored **CapabilityProfile** (what a container may do) is enforced at runtime by
**KubeArmor**, using the **ModelArmor** pattern — which exists precisely because "AI/ML models
are essentially processes" and MCP servers are untrusted code pulled from registries
(npm/uvx/Docker). ModelArmor's threat model is exactly ours: prompt-injection reading
`/root/.aws/credentials`, an agent running `apk add nmap`, a supply-chain payload phoning
home, GPU cryptomining. KubeArmor blocks these at the kernel (BPF-LSM / AppArmor / SELinux)
with no code changes.

### Authoring surface: CapabilityProfile

You write what the MCP container may do; everything else is denied:

```
container: grafana
allow:
  exec: [ /usr/bin/supergateway, /app/grafana-mcp ]
  filesystem:
    - read:  /etc/grafana
    - write: repo:notes @ /vault     # a mounted Repo, read-write
  network: [ tcp ]                    # protocol-level; destinations via egress (below)
default: deny
```

### Compiles to a KubeArmorPolicy

MCPHub labels each MCP pod and emits a `KubeArmorPolicy` (no hand-writing required):

```yaml
apiVersion: security.kubearmor.com/v1
kind: KubeArmorPolicy
metadata: { name: mcp-grafana, namespace: mcphub }
spec:
  selector:
    matchLabels: { mcphub.io/connection: grafana }
  process:
    matchPaths:
      - path: /usr/bin/supergateway
      - path: /app/grafana-mcp
  file:
    matchDirectories:
      - dir: /vault/            # mounted Repo (rw)
        recursive: true
      - dir: /etc/grafana/
        recursive: true
        readOnly: true
  network:
    matchProtocols:
      - protocol: tcp
  action: Allow
```

### Default-deny posture

KubeArmor enforces least-permissive (deny what isn't allow-listed) **once at least one
`Allow` policy exists** — which the compiled profile provides. MCPHub sets the namespace to a
blocking posture and enables file/network visibility:

```bash
kubectl annotate ns mcphub \
  kubearmor-file-posture=block \
  kubearmor-network-posture=block \
  kubearmor-capabilities-posture=block --overwrite
kubectl annotate ns mcphub kubearmor-visibility="process,file,network" --overwrite
```

Now anything outside the profile (reading credentials, installing packages, raw sockets) is
blocked and alerted with `Policy Name: DefaultPosture`.

### What KubeArmor does and does not cover

- **Covers:** process-exec allow-list, file read/write paths, network **protocol** (tcp/udp/
  icmp/raw) and per-binary (`fromSource`) rules, capabilities — at runtime.
- **Does not cover destinations:** KubeArmor's network rules are protocol/binary-level, not
  IP/FQDN. Destination allow-lists (e.g. only `grafana.example.com:443`) still come from
  **Cilium FQDN policy** or **NetworkPolicy**. Use both: KubeArmor for "which binary may open
  a tcp socket," Cilium for "to where."

### Telemetry → MCPHub audit

KubeArmor streams allow/audit/block events (via the relay, or `karmor logs --json`). MCPHub
ingests these into its audit trail, so a blocked syscall on an MCP shows up alongside tool
calls and approvals.

### Optional: pre-deploy red-teaming

ModelArmor can adversarially test an MCP image before it goes live (e.g. injection /
exfiltration scenarios). MCPHub can gate "publish this MCP" on a red-team pass.

## Deploying KubeArmor

- **Kubernetes (default backend):** install the operator via Helm; it **auto-detects** the
  node LSM (BPF-LSM / AppArmor / SELinux):
  ```bash
  helm repo add kubearmor https://kubearmor.github.io/charts && helm repo update kubearmor
  helm upgrade --install kubearmor-operator kubearmor/kubearmor-operator \
    -n kubearmor --create-namespace
  kubectl apply -f https://raw.githubusercontent.com/kubearmor/KubeArmor/main/pkg/KubeArmorOperator/config/samples/sample-config.yml
  ```
  Components: operator, per-node daemonset (enforcement), relay (telemetry), controller.
- **Non-Kubernetes (SQLite/Postgres host):** KubeArmor **systemd mode** on the VM/bare-metal
  host; install the `.deb`/tarball, `systemctl start kubearmor`, and manage host rules with
  `karmor vm policy add` using `KubeArmorHostPolicy` (`nodeSelector.matchLabels`). This keeps
  the self-contained, off-cluster deployments jailed too.

## What tools we use (summary)

- **Transport bridge:** Supergateway (stdio → Streamable HTTP, no SSE).
- **Sandbox runtime:** gVisor (default) or Kata Containers; nsjail/bubblewrap off-cluster.
- **Capability enforcement:** KubeArmor (ModelArmor pattern) — process/file/network/capability
  allow-list at runtime, default-deny posture; the `CapabilityProfile` is the authoring surface.
- **Hardening:** Kubernetes `securityContext` (non-root, read-only rootfs, drop caps,
  no-new-privileges), seccomp, AppArmor/SELinux, cgroup limits; Kyverno to require it.
- **Network destinations:** Cilium FQDN/L7 policy or NetworkPolicy (KubeArmor handles
  protocol/binary, not destination IP/FQDN).
- **Filesystem control:** read-only rootfs + explicit (incl. Repo) mounts.
- **Filesystem control:** read-only rootfs + explicit (incl. Repo) mounts.
