const express = require('express');
const cors = require('cors');
const { v4: uuidv4 } = require('uuid');

const app = express();
const port = process.env.PORT || 3000;

// In-memory storage for clients and tokens (would be persistent in production)
const clients = new Map(); // client_id -> {client_secret, redirect_uris, ...}
const authCodes = new Map(); // code -> {client_id, user_id, expires_at}
const tokens = new Map(); // access_token -> {user_id, client_id, expires_at, refresh_token}
const refreshTokens = new Map(); // refresh_token -> {client_id, user_id, expires_at}

app.use(cors());
app.use(express.json());

// Mock user database
const users = new Map();
users.set('user1', { id: 'user1', email: 'test@example.com' });

/**
 * RFC 9728 - Protected Resource Metadata
 * Endpoint for unauthenticated clients to discover authorization server
 */
app.get('/.well-known/oauth-protected-resource', (req, res) => {
  res.json({
    resource: req.protocol + '://' + req.get('host') + '/mcp',
    authorization_servers: [
      req.protocol + '://' + req.get('host')
    ],
    resource_documentation: null,
    authorization_server_documentation: null
  });
});

/**
 * RFC 8414 - Authorization Server Metadata
 * Well-known endpoint for OAuth server configuration
 */
app.get('/.well-known/oauth-authorization-server', (req, res) => {
  res.json({
    issuer: req.protocol + '://' + req.get('host'),
    authorization_endpoint: req.protocol + '://' + req.get('host') + '/oauth/authorize',
    token_endpoint: req.protocol + '://' + req.get('host') + '/oauth/token',
    registration_endpoint: req.protocol + '://' + req.get('host') + '/oauth/register',
    code_challenge_methods_supported: ['S256'],
    response_types_supported: ['code'],
    grant_types_supported: ['authorization_code', 'refresh_token'],
    token_endpoint_auth_methods_supported: ['none']
  });
});

/**
 * RFC 7591 - Dynamic Client Registration
 * Register a new client (public client with no secret)
 */
app.post('/oauth/register', (req, res) => {
  const { redirect_uris, client_name } = req.body;
  
  // Validate required fields
  if (!redirect_uris || !Array.isArray(redirect_uris) || redirect_uris.length === 0) {
    return res.status(400).json({ error: 'invalid_redirect_uri' });
  }
  
  // Create client (public client - no secret)
  const clientId = uuidv4();
  const client = {
    client_id: clientId,
    client_name: client_name || 'MCP Client',
    redirect_uris: redirect_uris,
    token_endpoint_auth_method: 'none', // Public client
    client_secret: undefined // No secret for public clients
  };
  
  clients.set(clientId, client);
  
  res.status(201).json({
    client_id: clientId,
    client_secret: undefined,
    client_name: client.client_name,
    redirect_uris: client.redirect_uris
  });
});

/**
 * Authorization endpoint - redirects to Google login
 * This is a simplified version - in practice, this would redirect to Google's OAuth endpoint
 */
app.get('/oauth/authorize', (req, res) => {
  const { response_type, client_id, redirect_uri, scope, state, code_challenge, code_challenge_method } = req.query;
  
  // Validate parameters
  if (response_type !== 'code') {
    return res.status(400).json({ error: 'unsupported_response_type' });
  }
  
  if (!client_id || !clients.has(client_id)) {
    return res.status(400).json({ error: 'invalid_client' });
  }
  
  if (!redirect_uri || !clients.get(client_id).redirect_uris.includes(redirect_uri)) {
    return res.status(400).json({ error: 'invalid_redirect_uri' });
  }
  
  if (code_challenge_method && code_challenge_method !== 'S256') {
    return res.status(400).json({ error: 'invalid_request' });
  }
  
  // In a real implementation, we would redirect to Google's OAuth endpoint here
  // For this POC, we'll simulate the authorization
  
  // Generate authorization code
  const code = uuidv4();
  authCodes.set(code, {
    client_id,
    user_id: 'user1', // Simulated user
    expires_at: Date.now() + 300000, // 5 minutes
    redirect_uri,
    scope,
    state,
    code_challenge,
    code_challenge_method
  });
  
  // Redirect back to client with authorization code
  const redirectUrl = new URL(redirect_uri);
  redirectUrl.searchParams.set('code', code);
  if (state) {
    redirectUrl.searchParams.set('state', state);
  }
  
  res.redirect(redirectUrl.toString());
});

/**
 * Token endpoint - exchange authorization code for access token
 */
app.post('/oauth/token', (req, res) => {
  const { grant_type, code, redirect_uri, client_id, code_verifier, refresh_token } = req.body;
  
  if (grant_type === 'authorization_code') {
    // Exchange authorization code for token
    if (!code) {
      return res.status(400).json({ error: 'invalid_request' });
    }
    
    const authCode = authCodes.get(code);
    if (!authCode) {
      return res.status(400).json({ error: 'invalid_grant' });
    }
    
    if (authCode.expires_at < Date.now()) {
      authCodes.delete(code);
      return res.status(400).json({ error: 'invalid_grant' });
    }
    
    if (client_id && authCode.client_id !== client_id) {
      return res.status(400).json({ error: 'invalid_grant' });
    }
    
    if (redirect_uri && authCode.redirect_uri !== redirect_uri) {
      return res.status(400).json({ error: 'invalid_grant' });
    }
    
    // Verify PKCE if used
    if (authCode.code_challenge) {
      if (!code_verifier) {
        return res.status(400).json({ error: 'invalid_request' });
      }
      
      // Simplified PKCE verification - in practice, would compute SHA-256
      if (code_verifier !== authCode.code_challenge) {
        return res.status(400).json({ error: 'invalid_grant' });
      }
    }
    
    // Generate tokens
    const accessToken = uuidv4();
    const refreshToken = uuidv4();
    
    const tokenData = {
      user_id: authCode.user_id,
      client_id: authCode.client_id,
      expires_at: Date.now() + 3600000, // 1 hour
      refresh_token: refreshToken
    };
    
    tokens.set(accessToken, tokenData);
    refreshTokens.set(refreshToken, {
      user_id: authCode.user_id,
      client_id: authCode.client_id,
      expires_at: Date.now() + 2592000000 // 30 days
    });
    
    // Clean up used authorization code
    authCodes.delete(code);
    
    res.json({
      access_token: accessToken,
      token_type: 'Bearer',
      expires_in: 3600,
      refresh_token: refreshToken
    });
  } else if (grant_type === 'refresh_token') {
    // Exchange refresh token for new access token
    if (!refresh_token) {
      return res.status(400).json({ error: 'invalid_request' });
    }
    
    const refreshTokenData = refreshTokens.get(refresh_token);
    if (!refreshTokenData) {
      return res.status(400).json({ error: 'invalid_grant' });
    }
    
    if (refreshTokenData.expires_at < Date.now()) {
      refreshTokens.delete(refresh_token);
      return res.status(400).json({ error: 'invalid_grant' });
    }
    
    // Rotate refresh token (delete old, create new)
    refreshTokens.delete(refresh_token);
    
    const newAccessToken = uuidv4();
    const newRefreshToken = uuidv4();
    
    const tokenData = {
      user_id: refreshTokenData.user_id,
      client_id: refreshTokenData.client_id,
      expires_at: Date.now() + 3600000, // 1 hour
      refresh_token: newRefreshToken
    };
    
    tokens.set(newAccessToken, tokenData);
    refreshTokens.set(newRefreshToken, {
      user_id: refreshTokenData.user_id,
      client_id: refreshTokenData.client_id,
      expires_at: Date.now() + 2592000000 // 30 days
    });
    
    res.json({
      access_token: newAccessToken,
      token_type: 'Bearer',
      expires_in: 3600,
      refresh_token: newRefreshToken
    });
  } else {
    res.status(400).json({ error: 'unsupported_grant_type' });
  }
});

/**
 * Middleware to authenticate MCP requests
 */
const authenticateMCP = (req, res, next) => {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    // According to RFC 9728, return WWW-Authenticate header with resource metadata
    return res.status(401).set('WWW-Authenticate', `Bearer realm="mcp", resource_metadata="${req.protocol}://${req.get('host')}/.well-known/oauth-protected-resource"`).json({
      error: 'unauthorized',
      message: 'Missing or invalid authorization header'
    });
  }
  
  const token = authHeader.substring(7);
  const tokenData = tokens.get(token);
  
  if (!tokenData) {
    return res.status(401).json({ error: 'invalid_token' });
  }
  
  if (tokenData.expires_at < Date.now()) {
    tokens.delete(token);
    return res.status(401).json({ error: 'invalid_token' });
  }
  
  req.user = users.get(tokenData.user_id);
  req.client_id = tokenData.client_id;
  next();
};

/**
 * MCP endpoint - protected resource
 */
app.post('/mcp', authenticateMCP, (req, res) => {
  // Simple echo implementation for demonstration
  res.json({
    jsonrpc: '2.0',
    id: req.body.id,
    result: {
      userInfo: req.user,
      method: req.body.method,
      params: req.body.params
    }
  });
});

/**
 * Simple demo tool endpoint
 */
app.post('/mcp/tools/demo', authenticateMCP, (req, res) => {
  res.json({
    id: req.body.id,
    result: {
      user: req.user,
      timestamp: new Date().toISOString(),
      message: 'Hello from authenticated MCP endpoint!'
    }
  });
});

app.listen(port, () => {
  console.log(`MCP Auth PoC server running at http://localhost:${port}`);
});