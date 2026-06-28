# Graph Report - mcphub  (2026-06-28)

## Corpus Check
- 87 files · ~58,417 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 714 nodes · 659 edges · 75 communities (68 shown, 7 thin omitted)
- Extraction: 99% EXTRACTED · 1% INFERRED · 0% AMBIGUOUS · INFERRED: 4 edges (avg confidence: 0.82)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6631a027`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Authentication Logic|Authentication Logic]]
- [[_COMMUNITY_Server Lifecycle Mocks|Server Lifecycle Mocks]]
- [[_COMMUNITY_Configuration Management|Configuration Management]]
- [[_COMMUNITY_MCP Proxy Server|MCP Proxy Server]]
- [[_COMMUNITY_MCP Server Implementation|MCP Server Implementation]]
- [[_COMMUNITY_Kubernetes Client Factory|Kubernetes Client Factory]]
- [[_COMMUNITY_E2E Test Utilities|E2E Test Utilities]]
- [[_COMMUNITY_Kubernetes Runtime Operations|Kubernetes Runtime Operations]]
- [[_COMMUNITY_Multi-Server Management|Multi-Server Management]]
- [[_COMMUNITY_Tool Registry|Tool Registry]]
- [[_COMMUNITY_MCP Server Controller|MCP Server Controller]]
- [[_COMMUNITY_Custom Resource Definitions|Custom Resource Definitions]]
- [[_COMMUNITY_Initialization Registry|Initialization Registry]]
- [[_COMMUNITY_OpenSpec Change Management|OpenSpec Change Management]]
- [[_COMMUNITY_MCP Core Types|MCP Core Types]]
- [[_COMMUNITY_Auth Error Handling|Auth Error Handling]]
- [[_COMMUNITY_CLI Entrypoint|CLI Entrypoint]]
- [[_COMMUNITY_Controller Test Suite|Controller Test Suite]]
- [[_COMMUNITY_E2E Test Suite|E2E Test Suite]]
- [[_COMMUNITY_Main Application Entry|Main Application Entry]]
- [[_COMMUNITY_Post-Install Script|Post-Install Script]]
- [[_COMMUNITY_MCP Shield Components|MCP Shield Components]]
- [[_COMMUNITY_Project Metadata|Project Metadata]]
- [[_COMMUNITY_OpenSpec Exploration|OpenSpec Exploration]]
- [[_COMMUNITY_Controller Unit Tests|Controller Unit Tests]]
- [[_COMMUNITY_API Group Versioning|API Group Versioning]]
- [[_COMMUNITY_Generated Deepcopy Code|Generated Deepcopy Code]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]

## God Nodes (most connected - your core abstractions)
1. `Executor — Complete Feature Specification` - 17 edges
2. `FAQ` - 13 edges
3. `Concepts` - 11 edges
4. `Custom Resources (CRDs)` - 11 edges
5. `Capability Enforcement (Design)` - 10 edges
6. `Catalog Features (Executor-inspired)` - 10 edges
7. `MCP Auth POC` - 10 edges
8. `Qwen MCP Auth PoC - Usage Guide` - 10 edges
9. `CLI` - 9 edges
10. `Authorization` - 9 edges

## Surprising Connections (you probably didn't know these)
- `openspec-apply-change` --conceptually_related_to--> `openspec-archive-change`  [INFERRED]
  .claude/skills/openspec-apply-change/SKILL.md → .claude/skills/openspec-archive-change/SKILL.md
- `MCP Shield Proxy` --conceptually_related_to--> `MCP Shield Controller`  [INFERRED]
  .references/mcpshield/README.md → .references/mcpshield/controller/README.md

## Import Cycles
- None detected.

## Communities (75 total, 7 thin omitted)

### Community 0 - "Authentication Logic"
Cohesion: 0.12
Nodes (20): app, authCodes, authenticate(), clients, handleMcp(), issueAuthCode(), ISSUER, issueTokens() (+12 more)

### Community 1 - "Server Lifecycle Mocks"
Cohesion: 0.10
Nodes (20): Acceptance Criteria Status, Cluster Deployment, Component Mapping, Local Validation, Next Steps for Production Implementation, Notion's exact AS hostname, advertised scopes, and RFC 8707 support are unverified, Open Questions Addressed, R0 — Streamable HTTP, not SSE (+12 more)

### Community 2 - "Configuration Management"
Cohesion: 0.10
Nodes (20): 10. Web UI & SDK, 11. Integrations Registry, 12. Deployment Targets (Apps), 13. Billing / Plans (cloud), 14. Testing, Emulators & Security Tooling, 15. Planned / Vision Features (vision.md), 16. Feature Summary (one-line index), 1. Core Concepts (+12 more)

### Community 3 - "MCP Proxy Server"
Cohesion: 0.12
Nodes (15): app, authCodes, clients, CLIENTS_FILE, ISSUER, issueTokens(), JWT_SECRET, logins (+7 more)

### Community 4 - "MCP Server Implementation"
Cohesion: 0.11
Nodes (17): 1. Well-Known Endpoints, 2. Dynamic Client Registration, 3. Authorization Flow, 4. Authenticated MCP Calls, Components Implemented, Deployment, Extending to Google OAuth, Getting Started (+9 more)

### Community 5 - "Kubernetes Client Factory"
Cohesion: 0.12
Nodes (15): CapabilityProfile, Connectors, Custom Resources (CRDs), GraphqlConnector, IdentityProvider, Managing users & roles: two ways, McpConnector, OAuthApp (+7 more)

### Community 6 - "E2E Test Utilities"
Cohesion: 0.12
Nodes (15): dependencies, cors, express, google-auth-library, openid-client, uuid, description, devDependencies (+7 more)

### Community 7 - "Kubernetes Runtime Operations"
Cohesion: 0.14
Nodes (13): app, authCodes, clients, ISSUER, issueTokens(), JWT_SECRET, logins, PORT (+5 more)

### Community 8 - "Multi-Server Management"
Cohesion: 0.14
Nodes (13): Approvals, Ask as an obligation, Authoring surfaces, Authorization, Engine: compiled to Cedar, Granularity, Policy — the default home, Resolution (+5 more)

### Community 9 - "Tool Registry"
Cohesion: 0.14
Nodes (13): Can I share an MCP that needs a credential without giving the credential away?, Do I have to use Kubernetes?, Do I have to write YAML?, Does it work for just me?, FAQ, How do I share with a team?, How is MCPHub different from connecting an MCP server directly?, Is it production ready? (+5 more)

### Community 10 - "MCP Server Controller"
Cohesion: 0.14
Nodes (13): 1. Start the server, 2. Configure Claude Code, 3. Launch Claude Code and authenticate, Acceptance Criteria, Architecture, Automated Tests, Configuration, Endpoints (+5 more)

### Community 11 - "Custom Resource Definitions"
Cohesion: 0.15
Nodes (12): Access & security, Appendices (design notes & research), Concepts & design, Documentation map, License, MCPHub Documentation, Operating, Project status (+4 more)

### Community 12 - "Initialization Registry"
Cohesion: 0.17
Nodes (11): dependencies, express, jose, @modelcontextprotocol/sdk, name, private, scripts, dev (+3 more)

### Community 13 - "OpenSpec Change Management"
Cohesion: 0.29
Nodes (3): openspec-apply-change, openspec-archive-change, OpenSpec Config

### Community 14 - "MCP Core Types"
Cohesion: 0.17
Nodes (11): Concepts, Connector, Credentials & accounts, Execution & approval, How they fit together, OAuth app, Policy, Service account & token (+3 more)

### Community 15 - "Auth Error Handling"
Cohesion: 0.17
Nodes (11): Architecture, Binaries, Components at a glance, Everything is API-driven, `mcphub` CLI, `mcphub` server (controller + API + proxy + UI), Request paths, Runtime backends (where MCP servers run) (+3 more)

### Community 16 - "CLI Entrypoint"
Cohesion: 0.17
Nodes (11): Adding a feature — pick the layer first, Checking it, Design, How "up" calls work (without breaking the rule), Project Structure, Proposed package tree, Spec, Success criteria (+3 more)

### Community 17 - "Controller Test Suite"
Cohesion: 0.17
Nodes (11): Changing a schema, CRD Lifecycle, Creation — schema authoring, Delivery — install, Design, Installing, Spec, Success criteria (+3 more)

### Community 18 - "E2E Test Suite"
Cohesion: 0.17
Nodes (11): dependencies, google-auth-library, jose, @modelcontextprotocol/sdk, name, private, scripts, dev (+3 more)

### Community 19 - "Main Application Entry"
Cohesion: 0.17
Nodes (11): dependencies, express, jose, @modelcontextprotocol/sdk, name, private, scripts, check (+3 more)

### Community 20 - "Post-Install Script"
Cohesion: 0.17
Nodes (4): clients, pendingCodes, server, tokens

### Community 22 - "Project Metadata"
Cohesion: 0.18
Nodes (10): Config knobs (per MCP), Container isolation ("tightly jailed"), Filesystem access (what it can see/write), Network egress control (what it can reach), Pod/container hardening (always on), Runtime & Isolation, Sandboxed runtime (kernel isolation), Transport: Streamable HTTP via Supergateway (+2 more)

### Community 23 - "OpenSpec Exploration"
Cohesion: 0.10
Nodes (19): Check for context, Ending Discovery, Guardrails, Handling Different Entry Points, OpenSpec Awareness, The Stance, What You Don't Have To Do, What You Might Do (+11 more)

### Community 24 - "Controller Unit Tests"
Cohesion: 0.18
Nodes (10): As a text file (GitOps), Attaching it to an MCP, Enforcement: Require vs Best effort, From the CLI, In the web UI, Sandbox & Access (Capability Profiles), The mental model, The status line (+2 more)

### Community 25 - "API Group Versioning"
Cohesion: 0.18
Nodes (10): Boundaries, Capability detection, Capability Enforcement (Design), Compilation & the reconcile loop, Enforcement planes and the v1 provider set, Extensibility, Fail-open vs fail-closed, Goals (+2 more)

### Community 26 - "Generated Deepcopy Code"
Cohesion: 0.18
Nodes (10): Add the URL — browser login, no token (recommended), Authentication, Connecting a client — one login for every MCP, Humans — OIDC (Google & Okta), Machines — fine-grained tokens (fallback), Machines — tokenless trust (preferred), Non-interactive (CI, agents), Or use the `mcphub` CLI bridge (+2 more)

### Community 27 - "Community 27"
Cohesion: 0.18
Nodes (10): 1. Unified tool catalog, 2. Source auto-detection, 3. Intent-based tool discovery, 4. Pause / resume executions (human-in-the-loop), 5. Policy enforcement, 6. Web UI, 7. First-run bootstrap token, Catalog Features (Executor-inspired) (+2 more)

### Community 28 - "Community 28"
Cohesion: 0.18
Nodes (9): app, authCodes, clients, cors, express, refreshTokens, tokens, users (+1 more)

### Community 29 - "Community 29"
Cohesion: 0.20
Nodes (9): A service account uses a token (machine identity), A user connects with Okta (OIDC, OAuth 2.0 authorization-code + PKCE), Dependency injection — Uber FX, Design, Operator framework — Kubebuilder, Spec, Technologies, The pool (+1 more)

### Community 30 - "Community 30"
Cohesion: 0.20
Nodes (9): Choosing and switching, Kubernetes / etcd (default), One store, including audit — no separate events database, Postgres, Secrets per backend, SQLite, Storage Backends, The three backends (+1 more)

### Community 31 - "Community 31"
Cohesion: 0.20
Nodes (9): Auth, CLI, Connectors, Embedded MCP Server, Output, Secrets, Sharing & policy, Tokens & service accounts (+1 more)

### Community 32 - "Community 32"
Cohesion: 0.20
Nodes (9): First access: the bootstrap token, How permissions resolve, Just-in-time access (optional), Label matching, Roles, Templating with traits, Two ways to manage (just like Teleport), Users (+1 more)

### Community 33 - "Community 33"
Cohesion: 0.20
Nodes (9): A. Structured read/query, B. Write / push with multiple writers, Concurrency across attached MCPs, How it maps to MCPHub, Model: Repo as a shared filesystem with attachable MCPs, Open questions, Repository Server (Obsidian vault as a structured API) — Design Note, The idea (+1 more)

### Community 34 - "Community 34"
Cohesion: 0.20
Nodes (9): 1. What we want, 2. What that looks like — Notion's auth, technically, 3. Goal, 4. Requirements, 5. Validation & deployment, 6. Acceptance criteria, 7. Open questions & gaps, 8. References (+1 more)

### Community 35 - "Community 35"
Cohesion: 0.22
Nodes (8): Beads - AI-Native Issue Tracking, Essential Commands, Get Started with Beads, Learn More, Quick Start, What is Beads?, Why Beads?, Working with Issues

### Community 36 - "Community 36"
Cohesion: 0.22
Nodes (8): Acceptance criteria status, Deploy (optional), How it works, MCP Auth POC — Notion-style browser login, federated to Google, Quick start — server + Claude Code + login, Run locally — no Google needed (dev IdP), Run with real Google, Verify without a client

### Community 37 - "Community 37"
Cohesion: 0.22
Nodes (8): 1. Run the control plane, 2. Add a credentialed connector (Grafana), 3. Share it with an Okta team, 4. Connect an agent, 5. Try a tool, see an approval, Getting Started, Next steps, Prerequisites

### Community 38 - "Community 38"
Cohesion: 0.22
Nodes (8): Access, Admin, Catalog, Design principle, Layout, Sign in, The token / identity builder, Web UI

### Community 39 - "Community 39"
Cohesion: 0.22
Nodes (8): Choosing a tier, Open design questions, Principle, Revocation, Sharing flow (Grafana → Okta team), end to end, Tier 0 — Tokenless (preferred), Tier 1 — Fine-grained token (fallback), Tokens & Identity

### Community 40 - "Community 40"
Cohesion: 0.22
Nodes (8): Audit, Credential isolation, Human-in-the-loop, Identity & least privilege, Reporting vulnerabilities, Security Model, Short-lived by default & revocation, Trust boundaries

### Community 41 - "Community 41"
Cohesion: 0.22
Nodes (8): Beyond, Phase 1.1 — Spec connectors (integrations), Phase 1 — Controller + per-MCP proxy, Phase 2 — Storage backends & authentication, Phase 3 — Authorization, Phase 4 — Tokenless identity & catalog, Phase 5 — Approvals & web UI, Roadmap

### Community 42 - "Community 42"
Cohesion: 0.22
Nodes (8): Commit & PR hygiene, Contributing, Conventions, Development setup, License, Proposing changes, Repository layout, Security disclosures

### Community 43 - "Community 43"
Cohesion: 0.22
Nodes (8): 1. Tailscale OAuth clients (token generation model), 2. Teleport (the tokenless / trusted-relationship model), 3. GitHub fine-grained tokens (the granularity bar), 4. The unifying principle, 5. Synthesis → the shipped design, Hermes — needs confirmation, Open questions, Token & Identity — Reference and Ideas

### Community 44 - "Community 44"
Cohesion: 0.22
Nodes (8): 2026-06-20T13:21:59-03:00 — docs · Getting Started · /#/getting-started, 2026-06-20T13:23:36-03:00 — mock · MCPs · /mockup.html, 2026-06-28T13:22:02-03:00 — docs · MCPHub Documentation · /#/?id=why-cedar-for-policies, 2026-06-28T13:24:05-03:00 — docs · MCPHub Documentation · /#/?id=why-cedar-for-policies, 2026-06-28T13:24:31-03:00 — docs · MCPHub Documentation · /#/?id=why-cedar-for-policies, 2026-06-28T14:46:53-03:00 — docs · MCPHub Documentation · /#/, 2026-06-28T14:47:34-03:00 — docs · MCPHub Documentation · /#/, MCPHub Review Comments

### Community 45 - "Community 45"
Cohesion: 0.22
Nodes (8): Deploying to the cluster (optional, only if local isn't enough), Files, mcp-auth-glm, Notes / out of scope (POC), Quick local smoke (no Google, no Claude Code), Run, Use it from Claude Code, What it is

### Community 46 - "Community 46"
Cohesion: 0.22
Nodes (8): dependencies, name, scripts, dev, start, test, type, version

### Community 47 - "Community 47"
Cohesion: 0.25
Nodes (7): Auth and permissions are plugins, not features, Checklist before you call it done, CLAUDE.md — Agent directives, Hooks for extension, Prime directive: decoupling, The core engine must run without auth or authz, Two teams, one contract

### Community 48 - "Community 48"
Cohesion: 0.25
Nodes (5): cb, challenge, code, loginId, verifier

### Community 49 - "Community 49"
Cohesion: 0.25
Nodes (7): Agents, Configuration essentials, Deployment, Kubernetes (team, native), Single host / team (Postgres), Solo / local (SQLite), Upgrades

### Community 50 - "Community 50"
Cohesion: 0.25
Nodes (3): authQ, challenge, verifier

### Community 51 - "Community 51"
Cohesion: 0.25
Nodes (5): DOCS, files, firstAnchor, leftover, parts

### Community 52 - "Community 52"
Cohesion: 0.25
Nodes (5): COMMENTS, HERE, MIME, ROOT, server

### Community 53 - "Community 53"
Cohesion: 0.29
Nodes (6): Beads, Core CLI Workflow, First Step, Preferred Route, Rules, What Belongs In Beads

### Community 54 - "Community 54"
Cohesion: 0.29
Nodes (6): Authentication, Lifecycle, Provider-specific cases: Google is not a special kind, Runtime contrast, Spec Connectors (OpenAPI & GraphQL), Versioning & refresh

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (6): Connector Extensibility (design note), Decision, Decompose "special" first, Options considered, Revisit triggers, The question

### Community 56 - "Community 56"
Cohesion: 0.29
Nodes (6): Comparison with a direct MCP connection, Discovery, Endpoints, Per-request flow, The MCP Proxy, Tool naming

### Community 57 - "Community 57"
Cohesion: 0.29
Nodes (6): Components, Implementation Plan, Manual Start, Quick Start with Task System, Qwen MCP Auth PoC, Requirements Covered

### Community 58 - "Community 58"
Cohesion: 0.43
Nodes (5): clearHover(), closePop(), context(), openPop(), save()

### Community 59 - "Community 59"
Cohesion: 0.33
Nodes (5): API, Authentication, Shape, Two ways in, one model, What the API is for

### Community 60 - "Community 60"
Cohesion: 0.33
Nodes (5): claude, glm, gpt, kimi, qwen

### Community 61 - "Community 61"
Cohesion: 0.33
Nodes (5): CRD schemas, Install, Key values, mcphub Helm chart, What it deploys

### Community 62 - "Community 62"
Cohesion: 0.40
Nodes (4): Example shape, MCPHub — Core Requirements, R1 — Scales from one person to a whole team, R2 — Set up a credentialed MCP and share it with an IdP team (Okta)

### Community 63 - "Community 63"
Cohesion: 0.40
Nodes (4): How MCPHub differs, Primary inspirations, Prior Art & Inspirations, Related work

### Community 65 - "Community 65"
Cohesion: 0.40
Nodes (4): Executor UI — scrape notes (localhost:4789, v1.5.13 BETA), Functional takeaways for MCPHub mock, Layout, Routes / screens

### Community 66 - "Community 66"
Cohesion: 0.67
Nodes (3): base64url, crypto, runDemo()

## Knowledge Gaps
- **486 isolated node(s):** `DOCS`, `files`, `firstAnchor`, `parts`, `leftover` (+481 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **7 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What connects `DOCS`, `files`, `firstAnchor` to the rest of the system?**
  _486 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Authentication Logic` be split into smaller, more focused modules?**
  _Cohesion score 0.11666666666666667 - nodes in this community are weakly interconnected._
- **Should `Server Lifecycle Mocks` be split into smaller, more focused modules?**
  _Cohesion score 0.09523809523809523 - nodes in this community are weakly interconnected._
- **Should `Configuration Management` be split into smaller, more focused modules?**
  _Cohesion score 0.09523809523809523 - nodes in this community are weakly interconnected._
- **Should `MCP Proxy Server` be split into smaller, more focused modules?**
  _Cohesion score 0.11695906432748537 - nodes in this community are weakly interconnected._
- **Should `MCP Server Implementation` be split into smaller, more focused modules?**
  _Cohesion score 0.1111111111111111 - nodes in this community are weakly interconnected._
- **Should `Kubernetes Client Factory` be split into smaller, more focused modules?**
  _Cohesion score 0.125 - nodes in this community are weakly interconnected._