# tennon Helm chart

Deploys the Tennon control plane: one `tennon` server (controller + management API + MCP proxy
+ embedded web UI) plus the `tennon.io/v1alpha1` CRDs and the RBAC the controller needs to
reconcile connector workloads.

## Install

```bash
helm install tennon oci://ghcr.io/yurifrl/charts/tennon -n tennon --create-namespace \
  --set oidc.issuer=https://accounts.google.com --set oidc.clientId=tennon
```

(From a source checkout, use the chart path `./charts/tennon` instead of the OCI ref.)

Grab the one-time bootstrap admin token:

```bash
kubectl -n tennon logs deploy/tennon | grep -i bootstrap
```

## What it deploys

| Resource | Notes |
|----------|-------|
| Deployment | the `tennon` server; `replicaCount > 1` uses leader election |
| Service | `api` (8080, also serves the UI) and `proxy` (8081) ports |
| ClusterRole/Binding | manage `tennon.io` CRs, connector Pods/Deployments/Secrets/HPAs/NetworkPolicies |
| Role/Binding | leases for leader election |
| CRDs | `tennon.io/v1alpha1` kinds, gated by `crds.install` |
| Ingress | optional (`ingress.enabled`) |

## Key values

| Key | Default | Purpose |
|-----|---------|---------|
| `image.repository` / `image.tag` | `ghcr.io/yurifrl/tennon` / appVersion | server image |
| `store.backend` | `kubernetes` | `kubernetes` \| `postgres` \| `sqlite` |
| `store.existingSecret` / `store.postgresDsn` | `""` | postgres DSN (key `dsn`) |
| `oidc.issuer` / `oidc.clientId` | `""` / `tennon` | user SSO login |
| `ingress.enabled` / `ingress.host` | `false` / `tennon.example.com` | expose API+UI+proxy |
| `crds.install` / `crds.keep` | `true` / `true` | manage CRDs with the chart |
| `rbac.capabilityEnforcers` | `false` | grant Cilium/KubeArmor policy access for CapabilityProfiles |

## CRD schemas

The shipped CRDs use an open spec (`x-kubernetes-preserve-unknown-fields`) because real
structural schemas aren't generated from source yet — the controller is the schema authority
and can also self-install CRDs (`--set crds.install=false`). Replace the schema in
`templates/crds.yaml` once Go types generate validation.
