# Qwen MCP Auth PoC

Implementation of an authenticated MCP URL with OAuth login, following the specification in ../SPEC.md.

This is a proof of concept that demonstrates:
- Streamable HTTP MCP server
- OAuth 2.1 with a mock identity provider
- Dynamic client registration (or equivalent)
- PKCE flow with browser login
- Token-based authentication for MCP calls

## Components

Based on the Notion MCP implementation, this POC includes:

1. Protected Resource Metadata (RFC 9728)
2. Authorization Server Metadata (RFC 8414)
3. Dynamic Client Registration or equivalent
4. Authorization Code + PKCE flow
5. Token validation on each MCP call

## Quick Start with Task System

1. **Start the server using the task system:**
```bash
cd /Users/yuri/Workdir/Yuri/mcphub/hack/pocs/mcp-auth
task run:qwen
```

2. **Configure your MCP client** (e.g., Claude Code):
The server will be available at `http://localhost:3000/mcp`

3. **Launch Claude Code:**
Open Claude Code and access the MCP panel. The first time you try to use a tool from the authenticated server, it will trigger the browser login flow automatically.

4. **Authenticate:**
Follow the browser login flow. Once authenticated, your tokens will be cached and reused for subsequent tool calls.

## Manual Start

If you prefer to start the server manually:

```bash
cd qwen
npm install
npm start
```

## Implementation Plan

1. Create a basic Streamable HTTP MCP server
2. Implement OAuth 2.1 with Google
3. Handle well-known endpoint discovery
4. Implement browser login flow with PKCE
5. Add token validation middleware
6. Test end-to-end flow locally
7. Document deployment process (if needed)

## Requirements Covered

- R0: Streamable HTTP (not SSE)
- R1: URL-only client config
- R2: Browser login, then reuse
- R3: Real IdP (Google)
- R4: Standards-conformant discovery
- R5: One login across MCPs
- R6: Verified on each call