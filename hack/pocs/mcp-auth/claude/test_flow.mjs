// End-to-end flow as an MCP client would do it, against the dev IdP (no Google).
import crypto from 'node:crypto';
const B = 'http://localhost:8000';
const j = async r => { const t = await r.text(); try { return JSON.parse(t); } catch { return t; } };

// 1. DCR
const reg = await j(await fetch(`${B}/register`, { method: 'POST', headers: { 'content-type': 'application/json' },
  body: JSON.stringify({ client_name: 'cli', redirect_uris: ['http://localhost:9999/cb'], token_endpoint_auth_method: 'none', grant_types: ['authorization_code', 'refresh_token'], response_types: ['code'] }) }));
const clientId = reg.client_id;

// 2. PKCE + authorize -> dev login page
const verifier = crypto.randomBytes(32).toString('base64url');
const challenge = crypto.createHash('sha256').update(verifier).digest('base64url');
const authUrl = `${B}/authorize?response_type=code&client_id=${clientId}&redirect_uri=${encodeURIComponent('http://localhost:9999/cb')}&code_challenge=${challenge}&code_challenge_method=S256&state=xyz&scope=email`;
const a = await fetch(authUrl, { redirect: 'manual' });
const loginId = new URL(a.headers.get('location'), B).searchParams.get('login');

// 3. submit the fake login form
const login = await fetch(`${B}/dev/login`, { method: 'POST', headers: { 'content-type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({ login: loginId, email: 'alice@example.com' }), redirect: 'manual' });
const cb = new URL(login.headers.get('location'));
const code = cb.searchParams.get('code');
if (cb.searchParams.get('state') !== 'xyz') throw new Error('state mismatch');

// 4. exchange code -> tokens
const tok = await j(await fetch(`${B}/token`, { method: 'POST', headers: { 'content-type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({ grant_type: 'authorization_code', code, code_verifier: verifier, client_id: clientId, redirect_uri: 'http://localhost:9999/cb' }) }));
if (!tok.access_token) throw new Error('no token: ' + JSON.stringify(tok));

// 5. call /mcp whoami with the bearer token, twice (reuse)
async function whoami(token) {
  const r = await fetch(`${B}/mcp`, { method: 'POST',
    headers: { 'content-type': 'application/json', accept: 'application/json, text/event-stream', authorization: `Bearer ${token}` },
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name: 'whoami', arguments: {} } }) });
  const text = await r.text();
  const m = text.match(/data: (\{.*\})/) || [null, text];
  return JSON.parse(m[1]).result.content[0].text;
}
console.log('call 1:', await whoami(tok.access_token));
console.log('call 2 (reuse):', await whoami(tok.access_token));

// unauthenticated is rejected
const no = await fetch(`${B}/mcp`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: '{}' });
console.log('no-token status:', no.status);
console.log('FLOW OK');
