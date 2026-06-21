# MCPHub Review Comments

Append-only log. Each entry: timestamp, surface, page, the original text, and the comment.

## 2026-06-20T13:21:59-03:00 — docs · Getting Started · /#/getting-started

**Original:**

> make install            # apply CRDs
> make deploy IMG=…       # deploy the controller
> Copy to clipboard
> Error
> Copied

**Comment:** cant use make this needs to work without the repo, needs to be self contained

---

## 2026-06-20T13:23:36-03:00 — mock · MCPs · /mockup.html

**Original:**

> NAME	TRANSPORT	TOOLS	TAGS	STATUS
> github	stdio	18 / 42	code prod	Ready
> jira	streamable-http	12 / 18	tickets	Ready
> slack	streamable-http	9 / 9	chat	Ready
> weather	sse	3 / 3	public	Ready
> internal-deploy	stdio	4 / 4	infra sensitive	Degraded

**Comment:** need to account for the case where the mcp dont have a image, is just a npm lib or uv

---
