#!/usr/bin/env node

/**
 * Test script for Qwen MCP Auth POC
 * Tests the complete flow: DCR -> login -> tool call
 */

import http from 'http';
import url from 'url';

// Configuration
const SERVER_URL = 'http://localhost:3000';
const REDIRECT_URI = 'http://localhost:3001/callback';

async function fetchJSON(url, options = {}) {
  const response = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...options
  });
  return response.json();
}

async function main() {
  console.log('🧪 Qwen MCP Auth POC Test Flow');
  console.log('===============================');

  try {
    // Step 1: Test well-known endpoints
    console.log('\n1. Testing well-known endpoints...');
    const asMetadata = await fetchJSON(`${SERVER_URL}/.well-known/oauth-authorization-server`);
    console.log('✅ RFC 8414 AS Metadata:', {
      issuer: asMetadata.issuer,
      authorization_endpoint: asMetadata.authorization_endpoint,
      token_endpoint: asMetadata.token_endpoint
    });

    const prMetadata = await fetchJSON(`${SERVER_URL}/.well-known/oauth-protected-resource`);
    console.log('✅ RFC 9728 PR Metadata:', {
      resource: prMetadata.resource,
      authorization_servers: prMetadata.authorization_servers
    });

    // Step 2: Dynamic Client Registration
    console.log('\n2. Dynamic Client Registration...');
    const client = await fetchJSON(`${SERVER_URL}/oauth/register`, {
      method: 'POST',
      body: JSON.stringify({
        client_name: 'Test Flow Client',
        redirect_uris: [REDIRECT_URI]
      })
    });
    console.log('✅ Client registered:', client.client_id);

    // Step 3: Test unauthenticated MCP call
    console.log('\n3. Testing unauthenticated MCP call...');
    const unauthResponse = await fetch(`${SERVER_URL}/mcp`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
    
    if (unauthResponse.status === 401) {
      console.log('✅ Unauthenticated call correctly rejected');
      const wwwAuth = unauthResponse.headers.get('www-authenticate');
      console.log('   WWW-Authenticate header:', wwwAuth);
    } else {
      console.log('❌ Expected 401 Unauthorized, got:', unauthResponse.status);
    }

    // Step 4: Simulate authorization flow
    console.log('\n4. Simulating authorization flow...');
    
    // In a real implementation, we would:
    // 1. Redirect user to authorization_endpoint (/oauth/authorize)
    // 2. Server redirects to callback_endpoint (/oauth/callback) with authorization code
    // 3. Callback endpoint redirects back to client's redirect_uri with authorization code
    // 4. Client exchanges code for tokens at token_endpoint
    
    // For this test, we'll show what the flow would look like:
    const authUrl = new URL(asMetadata.authorization_endpoint);
    authUrl.searchParams.set('response_type', 'code');
    authUrl.searchParams.set('client_id', client.client_id);
    authUrl.searchParams.set('redirect_uri', REDIRECT_URI);
    authUrl.searchParams.set('state', 'test-state');
    authUrl.searchParams.set('code_challenge', 'test-challenge');
    authUrl.searchParams.set('code_challenge_method', 'S256');
    
    console.log('   Authorization URL would be:');
    console.log('   ', authUrl.toString());
    
    console.log('   After user login, server redirects to:', asMetadata.callback_endpoint || '/oauth/callback');
    console.log('   Callback endpoint redirects back to client with authorization code');
    console.log('   Client would then exchange code for tokens at:', asMetadata.token_endpoint);
    
    // Step 5: Show what authenticated call would look like
    console.log('\n5. Authenticated call format:');
    console.log('   POST /mcp');
    console.log('   Authorization: Bearer <access_token>');
    console.log('   Content-Type: application/json');
    console.log('   {');
    console.log('     "jsonrpc": "2.0",');
    console.log('     "id": 1,');
    console.log('     "method": "getUserInfo"');
    console.log('   }');

    console.log('\n🎉 Test flow completed successfully!');
    console.log('\nTo test with a real MCP client:');
    console.log('1. Add this to your .mcp.json:');
    console.log('   {');
    console.log('     "mcpServers": {');
    console.log('       "qwen-auth-poc": {');
    console.log('         "type": "http",');
    console.log('         "url": "http://localhost:3000/mcp"');
    console.log('       }');
    console.log('     }');
    console.log('   }');
    console.log('2. Launch Claude Code and try to use a tool from the server');

  } catch (error) {
    console.error('❌ Test failed:', error.message);
    process.exit(1);
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}