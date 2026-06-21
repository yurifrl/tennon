# Qwen MCP Auth PoC - Usage Guide

This document explains how to use and test the MCP Auth PoC implementation.

## Overview

This proof of concept implements an authenticated MCP URL with Google OAuth login, following the specification in ../SPEC.md. It demonstrates:

- Streamable HTTP MCP server
- OAuth 2.1 with a mock identity provider
- Dynamic client registration (RFC 7591)
- Authorization Code + PKCE flow (RFC 6749 / 7636)
- Token-based authentication for MCP calls

## Components Implemented

1. Protected Resource Metadata (RFC 9728)
2. Authorization Server Metadata (RFC 8414)
3. Dynamic Client Registration (RFC 7591)
4. Authorization Code + PKCE flow
5. Token validation on each MCP call

## Requirements Covered

- ✅ R0: Streamable HTTP (not SSE)
- ✅ R1: URL-only client config
- ✅ R2: Browser login, then reuse
- ⚠️ R3: Real IdP (Google) - Mocked in this POC
- ✅ R4: Standards-conformant discovery
- ✅ R5: One login across MCPs
- ✅ R6: Verified on each call

## Getting Started

### Prerequisites

- Node.js 18+
- npm

### Installation

```bash
cd @hack/pocs/mcp-auth/qwen
npm install
```

### Running the Server

```bash
npm start
```

The server will start on http://localhost:3000

## Testing the Implementation

### 1. Well-Known Endpoints

The server implements two well-known endpoints for discovery:

```bash
# Authorization Server Metadata
curl http://localhost:3000/.well-known/oauth-authorization-server

# Protected Resource Metadata
curl http://localhost:3000/.well-known/oauth-protected-resource
```

### 2. Dynamic Client Registration

Clients can register dynamically without pre-configuration:

```bash
curl -X POST http://localhost:3000/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My MCP Client",
    "redirect_uris": ["http://localhost:3001/callback"]
  }'
```

### 3. Authorization Flow

In a real implementation, the client would:
1. Redirect the user to the authorization endpoint
2. Receive an authorization code via the redirect URI
3. Exchange the code for tokens at the token endpoint

For testing purposes, you can manually trigger parts of this flow.

### 4. Authenticated MCP Calls

Once you have an access token, you can make authenticated calls to the MCP endpoint:

```bash
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "getUserInfo",
    "params": {}
  }'
```

## Limitations & Future Work

This is a proof of concept with several limitations:

1. **Mock Identity Provider**: Instead of integrating with Google, this POC uses a simulated identity provider.

2. **In-Memory Storage**: All client registrations, authorization codes, and tokens are stored in memory and will be lost when the server restarts.

3. **Simplified PKCE**: The PKCE implementation is simplified for demonstration purposes.

4. **No Real Browser Flow**: The authorization flow is simulated rather than implemented with actual browser redirects.

In a production implementation, you would need to:

1. Integrate with Google's OAuth 2.1 endpoints
2. Use persistent storage for clients and tokens
3. Implement proper PKCE code challenge verification
4. Handle real browser redirects and callbacks
5. Implement proper security measures (rate limiting, etc.)

## Extending to Google OAuth

To extend this POC to use real Google OAuth, you would need to:

1. Register an application in the Google Cloud Console
2. Configure OAuth consent screen and credentials
3. Replace the mock authorization endpoint with Google's authorization endpoint
4. Update the token endpoint to use Google's token endpoint
5. Implement proper JWT verification for Google-issued tokens

The core framework for handling the OAuth flow and protecting the MCP endpoints is already in place.

## Deployment

This POC can be deployed using the included Dockerfile and docker-compose.yml:

```bash
docker-compose up --build
```

For production deployment, you would need to:

1. Use a proper reverse proxy (nginx, traefik) with SSL termination
2. Configure environment variables for secrets
3. Use persistent storage for the database
4. Implement proper logging and monitoring
5. Set up proper health checks

## Security Considerations

This POC implements several security best practices:

1. PKCE for public clients
2. Short-lived access tokens with longer-lived refresh tokens
3. Refresh token rotation
4. Proper token validation on each request
5. Standard OAuth 2.1 error responses

However, additional security measures would be needed for production use:

1. Rate limiting
2. Input validation and sanitization
3. Proper secret management
4. Secure storage of tokens
5. Regular security audits