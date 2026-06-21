#!/usr/bin/env node
// Manual test script for the full OAuth flow
// Run: node manual-test.mjs

import crypto from 'crypto';

const BASE_URL = process.env.BASE_URL || 'http://localhost:8000';

console.log('=== MCP Auth POC - Manual Test Script ===\n');
console.log(`Base URL: ${BASE_URL}\n`);

// Step 1: Check well-known endpoints
console.log('Step 1: Discovery');
console.log(`curl ${BASE_URL}/.well-known/oauth-protected-resource`);
console.log(`curl ${BASE_URL}/.well-known/oauth-authorization-server\n`);

// Step 2: Register a client
console.log('Step 2: Client Registration');
console.log(`curl -X POST ${BASE_URL}/oauth/register \\\n  -H "Content-Type: application/json" \\\n  -d '{"redirect_uris":["http://localhost/callback"]}'\n`);

// Step 3: Authorization (manual step - opens browser)
console.log('Step 3: Authorization (manual)');
const codeVerifier = crypto.randomBytes(32).toString('base64url');
const codeChallenge = crypto.createHash('sha256').update(codeVerifier).digest('base64url');
console.log(`PKCE Code Verifier: ${codeVerifier.slice(0, 20)}...`);
console.log(`PKCE Code Challenge: ${codeChallenge.slice(0, 20)}...\n`);
console.log(`Open this URL in your browser (replace <CLIENT_ID>):`);
console.log(`${BASE_URL}/oauth/authorize?response_type=code&client_id=<CLIENT_ID>&redirect_uri=http://localhost/callback&code_challenge=${codeChallenge}&code_challenge_method=S256&state=test123\n`);

// Step 4: Exchange code for token
console.log('Step 4: Token Exchange (after getting code from callback)');
console.log(`curl -X POST ${BASE_URL}/oauth/token \\\n  -H "Content-Type: application/x-www-form-urlencoded" \\\n  -d "grant_type=authorization_code" \\\n  -d "code=<CODE>" \\\n  -d "client_id=<CLIENT_ID>" \\\n  -d "redirect_uri=http://localhost/callback" \\\n  -d "code_verifier=${codeVerifier}"\n`);

// Step 5: Call MCP
console.log('Step 5: MCP Call');
console.log(`curl -X POST ${BASE_URL}/mcp \\\n  -H "Authorization: Bearer <TOKEN>" \\\n  -H "Content-Type: application/json" \\\n  -H "Accept: application/json, text/event-stream" \\\n  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'\n`);

console.log(`curl -X POST ${BASE_URL}/mcp \\\n  -H "Authorization: Bearer <TOKEN>" \\\n  -H "Content-Type: application/json" \\\n  -H "Accept: application/json, text/event-stream" \\\n  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"whoami"},"id":2}'\n`);
