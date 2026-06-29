# Tennon Review Comments

Append-only log. Each entry: timestamp, surface, page, the original text, and the comment.

## 2026-06-20T13:21:59-03:00 — docs · Getting Started · /#/getting-started

- **state:** done
- **type:** content
- **params:** {}

**Original:**

> make install            # apply CRDs
> make deploy IMG=…       # deploy the controller
> Copy to clipboard
> Error
> Copied

**Comment:** cant use make this needs to work without the repo, needs to be self contained

---

## 2026-06-20T13:23:36-03:00 — mock · MCPs · /mockup.html

- **state:** done
- **type:** feature
- **params:** {}

**Original:**

> NAME	TRANSPORT	TOOLS	TAGS	STATUS
> github	stdio	18 / 42	code prod	Ready
> jira	streamable-http	12 / 18	tickets	Ready
> slack	streamable-http	9 / 9	chat	Ready
> weather	sse	3 / 3	public	Ready
> internal-deploy	stdio	4 / 4	infra sensitive	Degraded

**Comment:** need to account for the case where the mcp dont have a image, is just a npm lib or uv

---

## 2026-06-28T13:22:02-03:00 — docs · Tennon Documentation · /#/?id=why-cedar-for-policies

- **state:** done
- **type:** link
- **params:** {}

**Original:**

> It gives RBAC now, ABAC (conditions) later, and analyzability to catch over-broad grants — all in-process, with no extra stateful service, preserving the embeddable SQLite story. You author Policy; Cedar is the compilation target. See Authorization.

**Comment:** add a link to what ceder is

---

## 2026-06-28T13:24:05-03:00 — docs · Tennon Documentation · /#/?id=why-cedar-for-policies

- **state:** done
- **type:** content
- **params:** {}

**Original:**

> Roadmap

**Comment:** Phase 1 is registring mcps and being able to add then in agents and agents being able to use then mcps with credentials

---

## 2026-06-28T13:24:31-03:00 — docs · Tennon Documentation · /#/?id=why-cedar-for-policies

- **state:** done
- **type:** content
- **params:** {}

**Original:**

> Phase 1 — Controller + per-MCP proxy

**Comment:** add phase 1.1 same thing but for integrations, focus on google integration as proff of concept

---

## 2026-06-28T14:46:53-03:00 — docs · Tennon Documentation · /#/

- **state:** done
- **type:** 
- **params:** 

**Original:**

> apiVersion: tennon.io/v1alpha1
> kind: TennonConnector
> metadata:
>   name: github
>   namespace: tennon
>   labels:
>     tags: code
> spec:
>   description: "GitHub repos, issues, and pull requests."   # agent-visible context
>   runtime:
>     container:
>       image: ghcr.io/github/github-mcp-server:latest
>     # stdio:  { command: npx, args: ["-y", "@modelcontextprotocol/server-github"] }
>     # remote: { url: https://api.githubcopilot.com/mcp }
>   transport: streamable-http
>   scaling:
>     replicas: 1                  # or autoscale: { minReplicas, maxReplicas, targetCPUUtilizationPercent }
>   capabilityProfile: github-jail # container runtime only
>   tools:
>     exposePrefix: "github_"
>   access:                        # local sharing — who may use THIS connector
>     - subject:
>         group: "okta:dev"
>       tools: ["*"]
>   tags: ["code"]
> status:
>   phase: Ready                   # Pending | Ready | Degraded | Failed
>   discoveredTools: 30
>   url: /mcp/github
> Copy to clipboardErrorCopied

**Comment:** There should be no access logic here

---

## 2026-06-28T14:47:34-03:00 — docs · Tennon Documentation · /#/

- **state:** done
- **type:** 
- **params:** 

**Original:**

> apiVersion: tennon.io/v1alpha1
> kind: OpenApiConnector
> metadata:
>   name: grafana-read
>   namespace: tennon
>   labels:
>     tags: observability
> spec:
>   description: "Grafana dashboards, alerts, and datasources (read-only)."
>   specUrl: https://grafana.example.com/openapi.json
>   baseUrl: https://grafana.example.com/api    # overrides the spec's servers; optional
>   tools:
>     allow: ["get_*", "search_*", "list_*"]     # scope which operations become tools
>     exposePrefix: "grafana_"
>   access:
>     - subject:
>         group: "okta:observability"
>       tools: ["*"]
>   approvals:
>     require: ["delete_*"]
>   tags: ["observability", "read"]
> status:
>   phase: Ready
>   generatedTools: 14
>   specHash: "sha256:1f3c…"                     # tools cache key; changes when the spec changes
>   url: /mcp/grafana-read
> Copy to clipboardErrorCopied

**Comment:** No access here to coinecgors dont know about qccess

---
