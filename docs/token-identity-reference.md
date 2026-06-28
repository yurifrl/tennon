# Token & Identity — Reference and Ideas

Research notes plus design ideas for MCPHub's token generation and (ideally tokenless)
authentication. This is a reference + scratchpad, not a final design. It pulls from
Tailscale OAuth clients, Teleport's certificate/tokenless model, and GitHub fine-grained
tokens, then sketches how MCPHub could combine them.

Driving requirements (see [requirements.md](requirements.md)):
- **R1** — works for one person and a whole team.
- **R2** — set up a credentialed MCP (e.g. Grafana) and share it with an Okta team, without
  exposing the credential.
- **Goal** — use this with **Hermes** (see "Hermes" note below); prefer a **service account
  + trusted relationship** over a copied secret. A token, when used, should be at least as
  granular as a **GitHub fine-grained token**.

---

## 1. Tailscale OAuth clients (token generation model)

What to copy: the **OAuth 2.0 client-credentials → short-lived scoped token** flow, plus
**tags** as the unit of least privilege.

- An **OAuth client** = client ID + long-lived secret, **owned by the tailnet (org), not a
  user** — it keeps working when its creator leaves. (Good for team/R1.)
- The client exchanges its credentials at a token endpoint
  (`POST /api/v2/oauth/token`) for an **access token that expires in exactly 1 hour**.
  **No refresh token** — you just re-request before expiry. Tokens are bearer tokens.
- **Scopes are fine-grained and per-resource with read/write variants**
  (`devices:core:read` vs `devices:core`, `auth_keys`, `dns`, `routes`, ...). Each scope
  maps to an explicit allowlist of API endpoints. A token request can **narrow** to a
  subset of the client's scopes (and tags) at mint time.
- **Tags** are the authorization handle: a client scoped to `auth_keys`/`devices:core`
  must select tags; anything it provisions **assumes that tag's identity**, and ACL rules
  are keyed on the tag. `tagOwners` lets a parent tag own children so one client can manage
  many tags.
- The client secret can itself act as an auth key to provision **ephemeral** nodes
  (auto-removed when offline), **reusable**, **one-off**, or **pre-approved** keys.
- Security: revoking the client **immediately revokes all tokens it issued**; mint events
  are audit-logged with the client ID as actor. Only long-lived secret is the client
  secret (and even that can be replaced by federated OIDC workload identity).

**MCPHub takeaway:** the *issuance* path can mirror this — an org-owned client/service
account mints short-lived (≈1h) bearer tokens, scoped down at mint time, with **tags** as
the sharing primitive that maps cleanly onto our existing MCP/bundle tags.

---

## 2. Teleport (the tokenless / trusted-relationship model)

What to copy: **no shared secret to leak**. Identity is proven via a trusted third party,
and access is a **short-lived certificate**, not a stored token.

- Core model: a built-in **CA issues short-lived X.509 / SSH certs** (default user TTL
  ~12h, services renew much faster). **Expiry is the revocation mechanism**; for instant
  cutoff there is **session/identity locking**.
- **Machine ID / `tbot`**: a renewal bot runs next to a service and continuously fetches
  fresh short-lived certs (minutes-scale). The workload never holds a long-lived credential.
- **Join methods** are the key idea, split into:
  - **Secret-based** (a static join token) — discouraged, "prone to exfiltration."
  - **Delegated / tokenless** — the instance proves identity via its platform's
    attestation, so **the token name is not a secret**:
    - **AWS IAM**: instance signs an STS request; rule allows only a given account/ARN.
    - **GitHub/GitLab/CircleCI**: the platform's **OIDC token** is verified against the
      provider's JWKS; allow-rules match `repository`, `sub`, `ref`.
    - **Kubernetes**: validates a ServiceAccount token via TokenReview / OIDC discovery.
    - **TPM**: hardware Endorsement Key — no shared secret at all.
  - Most delegated methods issue **non-renewable** certs (re-attest each time) — ideal for
    ephemeral CI/agents.
- **Workload Identity / SPIFFE**: issues `spiffe://<trust-domain>/<path>` IDs as **SVIDs**
  (X.509 or JWT); trust domain = the cluster's root of trust; SVIDs verified against the
  trust bundle; issued over a local Workload API after attestation.

**MCPHub takeaway:** the *strong* path is tokenless. A caller (Hermes, a CI agent, a
teammate's tool) presents a **platform-signed document** (OIDC JWT, K8s SA token, cloud
instance identity) that MCPHub validates against a configured **trust rule**, then issues a
**short-lived credential** for the proxy. No secret is stored on the client.

---

## 3. GitHub fine-grained tokens (the granularity bar)

What to copy: the **shape** of a token = `(resources) × (permissions × level) × expiry ×
owner`.

- Fine-grained PATs replace broad scopes with **50+ granular permissions**, each set to
  **none / read / write** (a few support **admin**); levels are hierarchical (write ⊇ read).
- **Per-resource selection**: confined to one owner; narrowed to all / selected / public
  repos. (One org per token.)
- **Expiration is first-class** (UI requires it; orgs can enforce a max lifetime).
- **Org approval policy**: tokens targeting an org can be created `pending` and only read
  public data until an admin approves; org owners can block fine-grained PATs entirely.
- A token never exceeds the issuing user's own access.

**MCPHub takeaway:** when a token *is* used, its grant must be expressible at this level:
specific MCPs/bundles/tags **×** specific tools **×** read/call/approve **×** expiry **×**
owner — which is exactly what the granular-token builder in the mockup and `Policy`
already aim at. Add **org approval** for sensitive grants and an **enforceable max
lifetime**.

---

## 4. The unifying principle

GitHub OIDC, SPIFFE/SPIRE, and Kubernetes bound tokens all converge:

> **Short-lived, audience-scoped, auto-rotated identity backed by a pre-established trust
> relationship — not a copied shared secret.** The verifier trusts the platform's signing
> keys; the workload proves *who/where it is*, not *what secret it holds*.

- **OIDC federation**: workload presents a JWT; verifier pre-registers (issuer, audience,
  subject-match) and validates the signature via the issuer's JWKS.
- **SPIFFE/SPIRE**: workload calls the Workload API with **no secret and without even
  knowing its own ID**, gets an SVID after attestation; X.509-SVIDs preferred over JWT
  (replay risk).
- **Kubernetes TokenRequest**: audience-bound, time-bound projected SA tokens, auto-rotated
  by the kubelet, validated against the cluster OIDC issuer.

---

## 5. Ideas for MCPHub (synthesis)

A tiered model — same system, scaling from one person (R1) to a team, with tokenless as the
preferred path:

### Tier 0 — Tokenless service account + trusted relationship (preferred, for Hermes)
- Define a **ServiceAccount** object (org-owned, Tailscale-style; survives its creator).
- Attach a **trust relationship** instead of a secret (Teleport-style): an OIDC/JWT trust
  rule (issuer + audience + subject claims), a K8s SA token rule, SPIFFE ID match, or a
  cloud instance-identity rule.
- At call time the caller presents its platform document; MCPHub validates it and **mints a
  short-lived (≈1h, non-renewable) proxy credential** scoped to that service account's
  policy. **Nothing to leak.** This is the Hermes path: Hermes runs with a workload
  identity, MCPHub trusts it, no token is copied around.

### Tier 1 — Fine-grained service token (fallback, GitHub-style)
- When tokenless isn't possible, mint a **fine-grained token** with: resources (MCPs /
  bundles / tags) × tools × level (call / read / require-approval) × **mandatory expiry** ×
  owner.
- **Tags are the sharing primitive** (Tailscale): grant `tag:observability` and it follows
  new Grafana-tagged MCPs automatically.
- Org/owner **approval policy** for sensitive grants; **enforceable max lifetime**;
  **revoke = kill all derived tokens immediately**; audit every mint.

### How R2 (Grafana → Okta team) plays out
- Owner registers the `grafana` MCP with a `secretKeyRef` credential (never exposed).
- Owner shares via `Policy` to the **Okta group** `observability` (resolved from the
  OIDC `groups` claim — no per-person edits; Okta membership drives access).
- A teammate authenticates via Okta (user OIDC), or **Hermes authenticates tokenless via
  its workload identity**; the proxy injects the owner's Grafana credential and enforces the
  group's tool grants + approvals. Members call Grafana tools; they never see the credential.

### Outcome / lifetime rules to standardize
- Default proxy credential TTL ~1h, **no refresh** (re-mint), revocation by expiry + an
  instant **lock** for emergencies (Teleport).
- Three decision outcomes preserved: allow / deny / require-approval (the approval obligation
  from the Cedar authorization design).

---

## Hermes — needs confirmation

"Use this with Hermes" is recorded as a goal, but I have **not confirmed which Hermes** is
meant (your own service/agent, a messaging gateway, an internal scheduler, etc.). The Tier-0
tokenless design above assumes Hermes is a **machine/workload that can carry a platform
identity** (OIDC/JWT, K8s SA, SPIFFE, or cloud instance identity) — which is what makes the
trusted-relationship, no-secret path work. If Hermes is something else (e.g. an end-user
chat surface), the integration shifts toward user OIDC + short-lived tokens instead. Please
confirm what Hermes is so this section can be made concrete.

---

## Open questions
1. Tokenless first, or fine-grained tokens first to ship? (Tier 0 vs Tier 1 priority.)
2. Short-lived **bearer tokens** (simpler) vs **mTLS certs** (Teleport-grade, more infra)?
3. Which trust relationships to support at launch: OIDC JWT, K8s SA token, SPIFFE, cloud
   instance identity?
4. Standard TTL and whether any long-lived token is ever allowed (with org max-lifetime).
