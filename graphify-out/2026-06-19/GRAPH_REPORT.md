# Graph Report - .  (2026-06-19)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 266 nodes · 381 edges · 27 communities (20 shown, 7 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 27 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

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
- [[_COMMUNITY_Controller Test Suite|Controller Test Suite]]
- [[_COMMUNITY_E2E Test Suite|E2E Test Suite]]
- [[_COMMUNITY_Post-Install Script|Post-Install Script]]
- [[_COMMUNITY_MCP Shield Components|MCP Shield Components]]
- [[_COMMUNITY_Project Metadata|Project Metadata]]
- [[_COMMUNITY_OpenSpec Exploration|OpenSpec Exploration]]

## God Nodes (most connected - your core abstractions)
1. `KubernetesRuntime` - 16 edges
2. `MCPServer` - 13 edges
3. `Run()` - 11 edges
4. `MCPServers` - 10 edges
5. `MockRuntime` - 10 edges
6. `Call` - 10 edges
7. `New()` - 9 edges
8. `Context` - 9 edges
9. `NewMCPServer()` - 8 edges
10. `NewProxy()` - 7 edges

## Surprising Connections (you probably didn't know these)
- `OpenSpec Config` --references--> `OPSX: Propose`  [INFERRED]
  openspec/config.yaml → .claude/commands/opsx/propose.md
- `StartServer()` --calls--> `NewKubernetesRuntimeFactoryWithKubeconfig()`  [INFERRED]
  .references/mcpshield/cmd/server/server.go → .references/mcpshield/pkg/runtime/kubernetes.go
- `main()` --calls--> `New()`  [INFERRED]
  .references/mcpshield/controller/cmd/main.go → .references/mcpshield/pkg/auth/auth.go
- `getMetricsOutput()` --calls--> `Run()`  [INFERRED]
  .references/mcpshield/controller/test/e2e/e2e_test.go → .references/mcpshield/controller/test/utils/utils.go
- `UncommentCode()` --calls--> `New()`  [INFERRED]
  .references/mcpshield/controller/test/utils/utils.go → .references/mcpshield/pkg/auth/auth.go

## Import Cycles
- None detected.

## Communities (27 total, 7 thin omitted)

### Community 0 - "Authentication Logic"
Cohesion: 0.10
Nodes (13): New(), TestAuth_CompleteFlow(), main(), Interface, Object, Principal, McpServer, McpServerList (+5 more)

### Community 1 - "Server Lifecycle Mocks"
Cohesion: 0.13
Nodes (14): Call, Controller, TestMCPServerLifecycle(), MockRuntimeFactoryMockRecorder, MockRuntimeMockRecorder, NewMockRuntime(), NewMockRuntimeFactory(), MockRuntime (+6 more)

### Community 2 - "Configuration Management"
Cohesion: 0.11
Nodes (13): APIConfig, AuthConfig, LogConfig, MCPServerConfig, APIConfig, AuthConfig, KubernetesConfig, LogConfig (+5 more)

### Community 3 - "MCP Proxy Server"
Cohesion: 0.15
Nodes (11): NewProxy(), MCPServers, Config, Config, Context, MCPRequest, MCPResponse, Request (+3 more)

### Community 4 - "MCP Server Implementation"
Cohesion: 0.14
Nodes (12): CancelFunc, InitializationRegistry, MCPServer, NewMCPServer(), Context, MCPRequest, MCPResponse, Runtime (+4 more)

### Community 5 - "Kubernetes Client Factory"
Cohesion: 0.22
Nodes (14): ClientConfig, Clientset, Runtime, RuntimeFactory, T, CreateKubernetesClient(), CreateKubernetesClientWithKubeconfig(), NewKubernetesRuntimeFactory() (+6 more)

### Community 6 - "E2E Test Utilities"
Cohesion: 0.21
Nodes (14): Cmd, getMetricsOutput(), tokenRequest, GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+6 more)

### Community 7 - "Kubernetes Runtime Operations"
Cohesion: 0.28
Nodes (4): KubernetesConfig, Config, Context, KubernetesRuntime

### Community 8 - "Multi-Server Management"
Cohesion: 0.17
Nodes (7): MCPServers, NewServers(), Config, Context, MCPRequest, MCPResponse, RuntimeFactory

### Community 9 - "Tool Registry"
Cohesion: 0.32
Nodes (3): NewToolRegistry(), ToolRegistry, RWMutex

### Community 10 - "MCP Server Controller"
Cohesion: 0.20
Nodes (7): Client, McpServerReconciler, Manager, Context, Request, Result, Scheme

### Community 11 - "Custom Resource Definitions"
Cohesion: 0.33
Nodes (7): ListMeta, ObjectMeta, McpServer, McpServerList, McpServerSpec, McpServerStatus, TypeMeta

### Community 12 - "Initialization Registry"
Cohesion: 0.43
Nodes (3): NewInitializationRegistry(), InitializationRegistry, MCPResponse

### Community 13 - "OpenSpec Change Management"
Cohesion: 0.29
Nodes (7): openspec-apply-change, openspec-archive-change, OpenSpec Config, openspec-propose, OPSX: Apply, OPSX: Archive, OPSX: Propose

### Community 14 - "MCP Core Types"
Cohesion: 0.40
Nodes (4): MCPRequest, MCPResponse, Runtime, RuntimeFactory

## Knowledge Gaps
- **60 isolated node(s):** `mcp-shield`, `Config`, `post-install.sh script`, `ObjectMeta`, `ListMeta` (+55 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **7 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMCPServer()` connect `MCP Server Implementation` to `Multi-Server Management`, `Tool Registry`, `Initialization Registry`, `Server Lifecycle Mocks`?**
  _High betweenness centrality (0.178) - this node is a cross-community bridge._
- **Why does `NewServers()` connect `Multi-Server Management` to `MCP Proxy Server`, `MCP Server Implementation`?**
  _High betweenness centrality (0.096) - this node is a cross-community bridge._
- **Why does `TestMCPServerLifecycle()` connect `Server Lifecycle Mocks` to `MCP Server Implementation`?**
  _High betweenness centrality (0.086) - this node is a cross-community bridge._
- **What connects `mcp-shield`, `Config`, `post-install.sh script` to the rest of the system?**
  _60 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Authentication Logic` be split into smaller, more focused modules?**
  _Cohesion score 0.09885057471264368 - nodes in this community are weakly interconnected._
- **Should `Server Lifecycle Mocks` be split into smaller, more focused modules?**
  _Cohesion score 0.12698412698412698 - nodes in this community are weakly interconnected._
- **Should `Configuration Management` be split into smaller, more focused modules?**
  _Cohesion score 0.10952380952380952 - nodes in this community are weakly interconnected._