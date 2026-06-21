/**
 * Simple client demo for the MCP Auth PoC
 * Demonstrates the complete flow:
 * 1. Dynamic client registration
 * 2. Authorization code flow with PKCE
 * 3. Token exchange
 * 4. Authenticated MCP call
 */

const crypto = require('crypto');
const base64url = require('base64url');

// In a real implementation, we would use a proper HTTP client library
// For this demo, we'll use the built-in fetch API (Node.js 18+)

// Configuration
const SERVER_URL = 'http://localhost:3000';
const REDIRECT_URI = 'http://localhost:3001/callback';

async function runDemo() {
  console.log('MCP Auth PoC Client Demo');
  console.log('========================');
  
  try {
    // Step 1: Dynamic Client Registration
    console.log('\n1. Registering client...');
    const registrationResponse = await fetch(`${SERVER_URL}/oauth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_name: 'MCP Auth Demo Client',
        redirect_uris: [REDIRECT_URI]
      })
    });
    
    if (!registrationResponse.ok) {
      throw new Error(`Registration failed: ${registrationResponse.status}`);
    }
    
    const clientCredentials = await registrationResponse.json();
    console.log('✓ Client registered successfully');
    console.log(`  Client ID: ${clientCredentials.client_id}`);
    
    // Step 2: Generate PKCE codes (simplified for demo)
    console.log('\n2. Generating PKCE codes...');
    const codeVerifier = base64url(crypto.randomBytes(32));
    const codeChallenge = codeVerifier; // Simplified - in practice, this would be a SHA-256 hash
    
    console.log('✓ PKCE codes generated');
    console.log(`  Code Verifier: ${codeVerifier}`);
    console.log(`  Code Challenge: ${codeChallenge}`);
    
    // Step 3: Authorization Request
    console.log('\n3. Simulating authorization request...');
    const state = 'random_state_string';
    const authUrl = new URL(`${SERVER_URL}/oauth/authorize`);
    authUrl.searchParams.set('response_type', 'code');
    authUrl.searchParams.set('client_id', clientCredentials.client_id);
    authUrl.searchParams.set('redirect_uri', REDIRECT_URI);
    authUrl.searchParams.set('state', state);
    authUrl.searchParams.set('code_challenge', codeChallenge);
    authUrl.searchParams.set('code_challenge_method', 'S256');
    
    console.log('  Authorization URL would be:');
    console.log(`  ${authUrl.toString()}`);
    
    // Step 4: Simulate receiving authorization code from redirect
    // In a real implementation, this would come from the redirect URI
    console.log('\n4. Simulating authorization code receipt...');
    const simulatedAuthCode = 'simulated_auth_code_from_redirect';
    console.log(`  Received authorization code: ${simulatedAuthCode}`);
    
    // Step 5: Token Request
    console.log('\n5. Exchanging authorization code for tokens...');
    const tokenResponse = await fetch(`${SERVER_URL}/oauth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        grant_type: 'authorization_code',
        code: simulatedAuthCode,
        redirect_uri: REDIRECT_URI,
        client_id: clientCredentials.client_id,
        code_verifier: codeVerifier
      })
    });
    
    if (!tokenResponse.ok) {
      const error = await tokenResponse.json();
      throw new Error(`Token exchange failed: ${error.error}`);
    }
    
    const tokens = await tokenResponse.json();
    console.log('✓ Tokens received successfully');
    console.log(`  Access Token: ${tokens.access_token.substring(0, 10)}...`);
    console.log(`  Refresh Token: ${tokens.refresh_token.substring(0, 10)}...`);
    console.log(`  Expires in: ${tokens.expires_in} seconds`);
    
    // Step 6: Authenticated MCP Call
    console.log('\n6. Making authenticated MCP call...');
    const mcpResponse = await fetch(`${SERVER_URL}/mcp`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${tokens.access_token}`
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: 1,
        method: 'getUserInfo',
        params: {}
      })
    });
    
    if (!mcpResponse.ok) {
      const error = await mcpResponse.json();
      throw new Error(`MCP call failed: ${error.error}`);
    }
    
    const mcpResult = await mcpResponse.json();
    console.log('✓ Authenticated MCP call successful');
    console.log('  User Info:', JSON.stringify(mcpResult.result.userInfo, null, 2));
    
    // Step 7: Demo Tool Call
    console.log('\n7. Calling demo tool...');
    const demoResponse = await fetch(`${SERVER_URL}/mcp/tools/demo`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${tokens.access_token}`
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: 2,
        method: 'demo',
        params: {}
      })
    });
    
    if (!demoResponse.ok) {
      const error = await demoResponse.json();
      throw new Error(`Demo tool call failed: ${error.error}`);
    }
    
    const demoResult = await demoResponse.json();
    console.log('✓ Demo tool call successful');
    console.log('  Result:', JSON.stringify(demoResult.result, null, 2));
    
    console.log('\n🎉 Demo completed successfully!');
    
  } catch (error) {
    console.error('❌ Demo failed:', error.message);
    process.exit(1);
  }
}

// Run the demo
if (require.main === module) {
  runDemo();
}