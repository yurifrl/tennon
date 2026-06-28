# Users & Roles

MCPHub manages **human identity and permissions the way [Teleport](https://goteleport.com)
does**: every user and role is a resource with the familiar `kind` / `metadata` / `spec`
envelope, stored in the backend and editable as YAML. There are two condition sets on every
role — `allow` and `deny` — and **deny always overrides allow**.

This page is the human-facing guide; the schemas live in [CRDs](crds.md#user). For machine
identities (CI, agents) use [`ServiceAccount`/`Token`](tokens-and-identity.md) instead.

## First access: the bootstrap token

On a fresh store there are no users yet, so the controller mints a one-time **bootstrap admin
token** and prints it on startup (see [Features §7](features.md)):

```
No users found — first-run bootstrap.
  → Open:  http://localhost:3000/?token=mh_boot_8f3c1d9a7be240xZ
  → Or run: mcphubctl open
```

Open that URL (or run `mcphubctl open`), then create your real account. The bootstrap token dies
the moment the first `User` exists, or after 1h.

## Users

A **User** is a human identity. It holds no permissions itself — access comes entirely from
the [`Role`](#roles)s in `spec.roles`. Users come in two flavors:

- **Local users** — created explicitly and stored in the backend:

  ```bash
  mcphubctl users add joe --roles=devtools-user
  # or, declaratively:
  mcphubctl create -f joe.yaml
  ```

- **SSO users** — created automatically at login from an [identity provider](authentication.md).
  Their OIDC claims populate `spec.traits` (notably `groups`), and roles are assigned by the
  identity provider's [`claimsToRoles`](crds.md#identityprovider) mapping.

Either way the user is a real backend object you can `mcphubctl get users`, edit, or **lock**:

```bash
mcphubctl users lock joe       # instantly cut off access without deleting the account
```

Locking terminates live sessions and blocks new ones — the emergency stop.

## Roles

A **Role** says what a subject may do, on two planes at once:

- **Control plane (`rules`)** — which MCPHub resources you may *manage*
  (`connector`, `role`, `user`, `token`, …) and with which verbs
  (`list`, `read`, `create`, `update`, `delete`, or `*`).
- **Data plane (`tools`)** — which tools you may *call*, by `tag` or `connector`,
  each with a tool glob. These grants compile to the **same rule primitive** as a
  [`Policy`](authorization.md#the-rule); a `Role` is just the Teleport-style packaging of
  them that also carries control-plane RBAC.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Role
metadata: { name: devtools-user }
spec:
  options: { maxSessionTtl: 8h }
  allow:
    rules:
      - { resources: [connector], verbs: [list, read] }
      - { resources: [token], verbs: [list, create, read, update, delete] }
    tools:
      - { connector: github, tools: ["*"] }
      - { tag: infra, tools: ["status"] }
    connectorLabels: { tags: ["code", "tickets"] }
  deny:
    tools:
      - { tools: ["*delete*"] }
```

### How permissions resolve

```
identity ──carries──▶ traits ──assigned──▶ roles ──contain──▶ rules + tool grants
```

1. An identity (local or SSO) carries **traits** — `spec.traits` for local users, OIDC
   claims for SSO users.
2. Traits map to **roles** (directly via `User.spec.roles`, or via the identity provider's
   claim-to-role mapping for SSO).
3. Roles contain **rules** (manage) and **tool grants** (call).
4. Every grant — from a role or a `Policy` — compiles to one
   rule primitive, and the call resolves by **most-restrictive-wins** (`block >
   ask > allow`). A `deny` is just a rule with `effect: block`, so "deny wins"
   is the top of that order. See [Authorization → Resolution](authorization.md#resolution)
   for the single algorithm.

### Templating with traits

Roles can interpolate the subject's traits, so one role serves many users:

```yaml
spec:
  allow:
    tools:
      - { tag: "{{external.groups}}", tools: ["*"] }   # SSO group claim
      # {{internal.<trait>}} pulls from a local user's spec.traits
```

### Label matching

`connectorLabels` selects connectors by label, like Teleport's `node_labels`: multiple
values under one key are **OR**, multiple keys are **AND**, and `'*': '*'` matches anything.
New connectors carrying a matching label are picked up automatically.

## Just-in-time access (optional)

A role can grant the right to *request* a more powerful role for a bounded window instead of
holding it permanently:

```yaml
spec:
  allow:
    request:        { roles: ["prod-admin"], maxDuration: 8h }   # may request
    reviewRequests: { roles: ["prod-admin"] }                    # may approve
```

A request becomes an `AccessRequest` that pauses until a reviewer approves; on approval the
subject assumes the role only for `maxDuration`, then drops back down. The built-in
`requester` and `reviewer` roles provide the basic flow.

## Two ways to manage (just like Teleport)

Identity is editable from either side; both converge on the backend store:

| Path | How | Best for |
|------|-----|----------|
| **CLI / API / UI** | `mcphubctl users add`, `mcphubctl create -f role.yaml`, web UI | day-to-day, ad-hoc |
| **Kubernetes Operator** | apply `User`/`Role` CRs; the controller reconciles them into the store | GitOps / Argo / Terraform |

On the cluster backend the **CRD *is* the resource**: `kubectl apply` a `Role` and it becomes
a live backend role; delete the CR and the role is removed. A finalizer blocks deletion until
the backend object is gone, and the `mcphub.io/keep: "true"` annotation keeps the backend
object when you remove the CR. This mirrors Teleport's `resources.teleport.dev` operator,
where `tctl`/Terraform and the Kubernetes Operator are two front doors to one backend.

See also: [Authentication](authentication.md) · [Authorization](authorization.md) ·
[Tokens & Identity](tokens-and-identity.md) · [CRDs](crds.md#user).
