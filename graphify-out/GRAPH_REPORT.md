# Graph Report - .  (2026-06-28)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 234 nodes · 228 edges · 28 communities (19 shown, 9 thin omitted)
- Extraction: 98% EXTRACTED · 2% INFERRED · 0% AMBIGUOUS · INFERRED: 5 edges (avg confidence: 0.9)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6453865f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]

## God Nodes (most connected - your core abstractions)
1. `TennonConnector CRD` - 6 edges
2. `rotateRefresh()` - 5 edges
3. `Tennon Core Requirements & Architecture` - 5 edges
4. `issueTokens()` - 4 edges
5. `scripts` - 4 edges
6. `MCP Proxy` - 4 edges
7. `closePop()` - 3 edges
8. `scripts` - 3 edges
9. `issueTokens()` - 3 edges
10. `scripts` - 3 edges

## Surprising Connections (you probably didn't know these)
- `Tennon Server` --references--> `MCP Proxy`  [INFERRED]
  cmd/tennon/main.go → docs/arch/07-proxy.md
- `Decoupling Directive` --references--> `Domain Ports`  [EXTRACTED]
  CLAUDE.md → domain/ports.go
- `MCP Proxy` --calls--> `Domain Ports`  [EXTRACTED]
  docs/arch/07-proxy.md → domain/ports.go
- `tctl CLI` --calls--> `MCP Proxy`  [EXTRACTED]
  cmd/tctl/main.go → docs/arch/07-proxy.md
- `Tennon Server` --implements--> `Domain Ports`  [INFERRED]
  cmd/tennon/main.go → domain/ports.go

## Import Cycles
- None detected.

## Communities (28 total, 9 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.12
Nodes (20): app, authCodes, authenticate(), clients, handleMcp(), issueAuthCode(), ISSUER, issueTokens() (+12 more)

### Community 1 - "Community 1"
Cohesion: 0.12
Nodes (15): app, authCodes, clients, CLIENTS_FILE, ISSUER, issueTokens(), JWT_SECRET, logins (+7 more)

### Community 2 - "Community 2"
Cohesion: 0.12
Nodes (18): Tennon Core Requirements & Architecture, Repository Server Design, Token & Identity Reference, Cedar Policy Engine, Tennon Helm Chart, Decoupling Directive, tctl CLI, Tennon Server (+10 more)

### Community 3 - "Community 3"
Cohesion: 0.12
Nodes (15): dependencies, cors, express, google-auth-library, openid-client, uuid, description, devDependencies (+7 more)

### Community 4 - "Community 4"
Cohesion: 0.14
Nodes (13): app, authCodes, clients, ISSUER, issueTokens(), JWT_SECRET, logins, PORT (+5 more)

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (11): dependencies, express, jose, @modelcontextprotocol/sdk, name, private, scripts, dev (+3 more)

### Community 6 - "Community 6"
Cohesion: 0.17
Nodes (11): dependencies, google-auth-library, jose, @modelcontextprotocol/sdk, name, private, scripts, dev (+3 more)

### Community 7 - "Community 7"
Cohesion: 0.17
Nodes (11): dependencies, express, jose, @modelcontextprotocol/sdk, name, private, scripts, check (+3 more)

### Community 8 - "Community 8"
Cohesion: 0.17
Nodes (4): clients, pendingCodes, server, tokens

### Community 9 - "Community 9"
Cohesion: 0.18
Nodes (9): app, authCodes, clients, cors, express, refreshTokens, tokens, users (+1 more)

### Community 10 - "Community 10"
Cohesion: 0.22
Nodes (8): dependencies, name, scripts, dev, start, test, type, version

### Community 11 - "Community 11"
Cohesion: 0.25
Nodes (5): cb, challenge, code, loginId, verifier

### Community 12 - "Community 12"
Cohesion: 0.25
Nodes (3): authQ, challenge, verifier

### Community 13 - "Community 13"
Cohesion: 0.25
Nodes (4): Bucket, buckets, DOCS, reg

### Community 14 - "Community 14"
Cohesion: 0.25
Nodes (5): COMMENTS, HERE, MIME, ROOT, server

### Community 15 - "Community 15"
Cohesion: 0.43
Nodes (5): clearHover(), closePop(), context(), openPop(), save()

### Community 16 - "Community 16"
Cohesion: 0.33
Nodes (5): claude, glm, gpt, kimi, qwen

### Community 18 - "Community 18"
Cohesion: 0.67
Nodes (3): base64url, crypto, runDemo()

### Community 19 - "Community 19"
Cohesion: 0.67
Nodes (3): Prior Art & Inspirations, Executor Feature Specification, Executor UI Scrape Notes

## Knowledge Gaps
- **129 isolated node(s):** `DOCS`, `buckets`, `Bucket`, `reg`, `HERE` (+124 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **9 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What connects `DOCS`, `buckets`, `Bucket` to the rest of the system?**
  _131 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.11666666666666667 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.11695906432748537 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.12418300653594772 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.125 - nodes in this community are weakly interconnected._
- **Should `Community 4` be split into smaller, more focused modules?**
  _Cohesion score 0.14285714285714285 - nodes in this community are weakly interconnected._