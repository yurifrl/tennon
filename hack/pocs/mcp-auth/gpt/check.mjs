import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import { readFileSync } from 'node:fs';
import { spawn } from 'node:child_process';

const PORT = 8899;
const BASE = `http://127.0.0.1:${PORT}`;
const CALLBACK = 'http://127.0.0.1/callback';

const server = spawn(process.execPath, ['server.js'], {
  cwd: new URL('.', import.meta.url),
  env: { ...process.env, PORT: String(PORT), PUBLIC_URL: BASE },
  stdio: ['ignore', 'pipe', 'pipe']
});

let logs = '';
server.stdout.on('data', d => { logs += d; });
server.stderr.on('data', d => { logs += d; });

async function waitForServer() {
  for (let i = 0; i < 50; i++) {
    if (server.exitCode !== null) throw new Error(`server exited early\n${logs}`);
    try {
      const r = await fetch(`${BASE}/.well-known/oauth-protected-resource`);
      if (r.ok) return;
    } catch {}
    await new Promise(r => setTimeout(r, 100));
  }
  throw new Error(`server did not start\n${logs}`);
}

async function req(path, options = {}) {
  const r = await fetch(`${BASE}${path}`, { redirect: 'manual', ...options });
  const text = await r.text();
  let body = text;
  try { body = text ? JSON.parse(text) : null; } catch {}
  return { status: r.status, headers: r.headers, body };
}

function challenge(verifier) {
  return crypto.createHash('sha256').update(verifier).digest('base64url');
}

try {
  assert.match(readFileSync(new URL('server.js', import.meta.url), 'utf8'), /\/oauth\/callback/);
  await waitForServer();

  const pr = await req('/.well-known/oauth-protected-resource');
  assert.equal(pr.status, 200);
  assert.equal(pr.body.resource, `${BASE}/mcp`);
  assert.deepEqual(pr.body.authorization_servers, [BASE]);

  const as = await req('/.well-known/oauth-authorization-server');
  assert.equal(as.status, 200);
  assert.ok(as.body.authorization_endpoint);
  assert.ok(as.body.token_endpoint);
  assert.ok(as.body.registration_endpoint);
  assert.ok(as.body.code_challenge_methods_supported.includes('S256'));

  const unauth = await req('/mcp', { method: 'POST', headers: { 'content-type': 'application/json' }, body: '{}' });
  assert.equal(unauth.status, 401);
  assert.match(unauth.headers.get('www-authenticate') ?? '', /resource_metadata=/);

  const reg = await req('/register', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ redirect_uris: [CALLBACK], token_endpoint_auth_method: 'none' })
  });
  assert.equal(reg.status, 201);
  assert.ok(reg.body.client_id);
  assert.equal(reg.body.client_secret, undefined);

  const verifier = crypto.randomBytes(32).toString('base64url');
  const auth = await req(`/authorize?${new URLSearchParams({
    response_type: 'code',
    client_id: reg.body.client_id,
    redirect_uri: CALLBACK,
    code_challenge: challenge(verifier),
    code_challenge_method: 'S256',
    state: 'abc',
    scope: 'openid email mcp'
  })}`);
  assert.equal(auth.status, 302);
  const login = auth.headers.get('location');
  assert.match(login, /^\/dev\/login\?/);
  const loginId = new URL(login, BASE).searchParams.get('login');

  const loginRes = await req(login, {
    method: 'POST',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ login: loginId, email: 'person@example.com' })
  });
  assert.equal(loginRes.status, 302);
  const back = new URL(loginRes.headers.get('location'));
  assert.equal(back.origin + back.pathname, CALLBACK);
  assert.equal(back.searchParams.get('state'), 'abc');

  const token = await req('/token', {
    method: 'POST',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      client_id: reg.body.client_id,
      code: back.searchParams.get('code'),
      redirect_uri: CALLBACK,
      code_verifier: verifier
    })
  });
  assert.equal(token.status, 200, JSON.stringify(token.body));
  assert.ok(token.body.access_token);
  assert.ok(token.body.refresh_token);

  const initialize = await req('/mcp', {
    method: 'POST',
    headers: {
      authorization: `Bearer ${token.body.access_token}`,
      'content-type': 'application/json',
      accept: 'application/json, text/event-stream'
    },
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '2025-06-18', capabilities: {}, clientInfo: { name: 'check', version: '1' } } })
  });
  assert.equal(initialize.status, 200, initialize.body);

  const tools = await req('/mcp', {
    method: 'POST',
    headers: {
      authorization: `Bearer ${token.body.access_token}`,
      'content-type': 'application/json',
      accept: 'application/json, text/event-stream'
    },
    body: JSON.stringify({ jsonrpc: '2.0', id: 2, method: 'tools/call', params: { name: 'whoami', arguments: {} } })
  });
  assert.equal(tools.status, 200, tools.body);
  assert.match(JSON.stringify(tools.body), /person@example\.com/);

  const later = await req('/mcp', {
    method: 'POST',
    headers: {
      authorization: `Bearer ${token.body.access_token}`,
      'content-type': 'application/json',
      accept: 'application/json, text/event-stream'
    },
    body: JSON.stringify({ jsonrpc: '2.0', id: 3, method: 'tools/list', params: {} })
  });
  assert.equal(later.status, 200, later.body);

  console.log('ALL PASS');
} finally {
  server.kill();
}
