# CRD Lifecycle

How MCPHub's `mcphub.io/v1alpha1` CRDs are **created** (schema authored), **delivered**
(installed), and **updated** (evolved) — without breaking the "one binary, any backend"
promise. For the schemas themselves, see [Custom Resources (CRDs)](crds.md).

This page follows the standard design-page shape: **Spec** (the rules), **Design** (how we
meet them), **Usage** (day to day).

---

## Spec

1. **The Go type is the source of truth.** CRD YAML is generated, never hand-written. Generated
   output and the types may not drift.
2. **Validation runs on every backend.** Rules must hold on Kubernetes *and* Postgres/SQLite.
   A check that only an apiserver enforces is not enough.
3. **Delivery is zero-dependency by default.** A fresh install needs no extra CLI, no GitOps
   tooling, no webhook cert plumbing.
4. **Schema evolution never loses data.** Stored objects survive an upgrade.

### Success criteria

- `make manifests && git diff --exit-code` is clean in CI (no drift).
- The control plane starts and serves CRs on all three backends with identical validation.
- A version bump (`v1alpha1` → `v1beta1`) leaves existing objects readable.

---

## Design

### Creation — schema authoring

Go types carry `+kubebuilder` markers; `controller-gen` generates the CRD YAML
(`config/crd/bases`) and deepcopy code. Validation and defaulting are expressed **in the
schema** wherever possible — `x-kubernetes-validations` (CEL) for rules, `default:` for
defaults (e.g. `scaling.replicas: 1`).

Anything CEL can't express — cross-object lookups, rules that consult the store — lives in
`domain`/`store` validation, **not** a webhook. This is deliberate: webhooks are a
Kubernetes-apiserver-only mechanism, so on the Postgres/SQLite backends there is no apiserver
to call them. Putting the logic in code is the only way it runs on every backend — which makes
an admission webhook largely redundant with code you must write anyway.

### Delivery — install

`kustomize` bundles CRDs + RBAC + Deployment into **one versioned release manifest**
(`mcphub.yaml`) — the same pattern Cluster API uses for its `components.yaml`. Two install
paths consume it:

- **Self-install (default, k8s backend).** The controller embeds the generated CRDs and applies
  them with **server-side apply** on boot, before starting its informers. Zero external
  dependency. The controller's RBAC needs `apiextensions` `create/update/patch` — **not**
  `delete` (dropping a CRD would delete every stored connector/policy).
- **GitOps / declarative.** Users apply the generated bundle (or the chart's CRDs) themselves.
  For teams that want cluster state fully declared in Git and `crds.install=false` to keep the
  controller out of the apiserver's schema.

On **Postgres/SQLite there is no CRD install step at all** — the schema is compiled into the
binary; the "CRD" is a code type backed by a table.

### Update — evolution

- **Additive change** (new optional field): regenerate, re-apply via server-side apply.
  Backward-compatible, no webhook.
- **Breaking change / new version:** introduce `v1beta1`, mark it the storage version, and add
  **hub-and-spoke conversion** — a conversion webhook on the Kubernetes backend, plain Go
  conversion functions in `store` everywhere else. This is the one place a webhook is genuinely
  required, and only once two versions are served simultaneously.

A CI drift gate (`make manifests && git diff --exit-code`) keeps the generated YAML in lockstep
with the Go types, so the embedded CRDs can never rot.

### What we deliberately do *not* build

Cluster API ships a dedicated CLI (`clusterctl`) and an inter-provider "API contract" because
it orchestrates many independent providers and bootstraps workload clusters from a management
cluster. MCPHub is a single control plane, so neither applies — `mcphub install` / self-install
is the right-sized equivalent. Argo CD's pure-declarative model (no self-install, controller
has zero CRD permissions) fits a Kubernetes-only tool; MCPHub also runs off-cluster, so
self-install is the better default.

> **ponytail:** webhooks stay out of the Helm chart until a real `v1beta1` exists. Everything
> before that is CEL-in-schema plus in-code validation, delivered as a generated bundle and
> self-installed.

---

## Usage

### Changing a schema

1. Edit the Go type and its `+kubebuilder` markers.
2. `make manifests` — regenerates `config/crd/bases` and the embedded copy.
3. Add/extend validation: CEL in the schema for simple rules; `domain`/`store` for the rest.
4. Commit the regenerated YAML. CI fails if you forgot (`git diff --exit-code`).

### Installing

```bash
# Self-install (default): just run the controller; it applies its embedded CRDs.
helm install mcphub ./charts/mcphub -n mcphub --create-namespace

# GitOps: let the chart manage CRDs declaratively, controller stays out of the schema.
helm install mcphub ./charts/mcphub --set crds.install=true --set crds.keep=true
# (or apply the generated bundle: kubectl apply -f mcphub.yaml)
```

> The chart currently ships **open-schema** CRDs (`x-kubernetes-preserve-unknown-fields`) as a
> placeholder until `controller-gen` exists in the repo — the same shortcut Argo CD's
> `Application` CRD ships in production. Replace with generated structural schemas once the Go
> types land, then flip `crds.install=false` to make self-install the default.
