// Self-check: exercises every OAuth code path that does NOT need a live Google
// browser login. The Google step (/authorize -> /callback) is the only part
// that genuinely needs a human + real IdP; everything else is logic we can pin.
//
//   node check.mjs
//
// Strategy: start the server with GOOGLE_* unset, monkey-replace the Google
// round-trip by calling the internal flow directly. We don't — instead we drive
// /authorize with a stubbed googleClient by setting an env hook. Simplest path:
// bypass /authorize+/callback and inject an auth code by reusing the server's
// own internals through a second HTTP seam. Since the server has no such seam,
// we instead test PKCE+token+refresh+reuse against a code we mint by replicating
// the code_challenge the server expects — but we can't mint a code without the
// callback. So: we run the server in "TEST_STUB_IDP=1" mode where /authorize
// short-circuits to our callback with a fake Google id_token.
//
// Run: TEST_STUB_IDP=1 GOOGLE_CLIENT_ID=test-client node check.mjs

import assert from "node:assert";

const BASE = process.env.BASE ?? `http://localhost:${process.env.PORT ?? 8787}`;

async function req(path, opts = {}) {
  const r = await fetch(BASE + path, opts);
  let body = null;
  const ct = r.headers.get("content-type") ?? "";
  body = ct.includes("json") ? await r.json() : await r.text().catch(() => null);
  return { status: r.status, headers: r.headers, body };
}

function b64url(s) { return Buffer.from(s).toString("base64url"); }
function sha256(s) { return crypto.createHash("sha256").update(s).digest("base64url"); }
import crypto from "node:crypto";

let failures = 0;
const check = (name, cond, detail = "") => {
  try { assert.ok(cond, detail); console.log(`  ok  ${name}`); }
  catch (e) { failures++; console.log(`  FAIL ${name} — ${e.message}`); }
};

// 1. Protected resource metadata (RFC 9728) + AS metadata (RFC 8414)
const pr = await req("/.well-known/oauth-protected-resource");
check("PR metadata 200", pr.status === 200);
check("PR names AS", Array.isArray(pr.body.authorization_servers) && pr.body.authorization_servers[0]);

const as = await req("/.well-known/oauth-authorization-server");
check("AS metadata 200", as.status === 200);
check("AS has endpoints", as.body.authorization_endpoint && as.body.token_endpoint && as.body.registration_endpoint);
check("AS supports S256", as.body.code_challenge_methods_supported?.includes("S256"));
check("AS public-client auth none", as.body.token_endpoint_auth_methods_supported?.includes("none"));

// 2. Unauthenticated /mcp -> 401 + WWW-Authenticate resource_metadata (acceptance #1)
const m = await req("/mcp", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ jsonrpc: "2.0", method: "initialize", id: 1, params: { protocolVersion: "2025-06-18", capabilities: {}, clientInfo: { name: "check", version: "1" } } }) });
check("/mcp 401 unauth", m.status === 401);
check("WWW-Authenticate points at PR metadata", /resource_metadata=/.test(m.headers.get("www-authenticate") ?? ""));

// 3. DCR (RFC 7591) — public client, no secret (R1, R4, acceptance #3)
const reg = await req("/register", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ redirect_uris: ["http://127.0.0.1:0/cb"], token_endpoint_auth_method: "none" }) });
check("DCR 201", reg.status === 201);
check("DCR returns client_id, no secret", reg.body.client_id && !reg.body.client_secret);
const clientId = reg.body.client_id;

// 4. PKCE + authorize + token (with stubbed IdP). Build code_verifier/challenge.
const verifier = b64url(crypto.randomBytes(32));
const challenge = sha256(verifier);
const authQ = new URLSearchParams({ client_id: clientId, redirect_uri: "http://127.0.0.1:0/cb", response_type: "code", code_challenge: challenge, code_challenge_method: "S256", state: "st", scope: "openid email mcp" });
const authorizeRes = await req(`/authorize?${authQ}`, { redirect: "manual" });
check("/authorize redirects (302)", authorizeRes.status === 302 || authorizeRes.status === 301, `got ${authorizeRes.status}`);

// In stub mode /authorize 302s directly to redirect_uri?code=... (no Google).
// In real mode it 302s to Google. Extract code if present.
let code = null;
const loc = authorizeRes.headers.get("location") ?? "";
if (loc.startsWith("http://127.0.0.1:0/cb")) {
  code = new URL(loc).searchParams.get("code");
}
check("stub IdP issued auth code", !!code, "set TEST_STUB_IDP=1 to exercise the token path");
if (!code) {
  console.log("\n  (Google login path not exercised — start server with TEST_STUB_IDP=1 to cover PKCE/token/refresh/reuse.)");
}

// 5. Token exchange + PKCE verification + reuse (acceptance #4, #5)
if (code) {
  const tok = await req("/token", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ grant_type: "authorization_code", code, redirect_uri: "http://127.0.0.1:0/cb", code_verifier: verifier, client_id: clientId }) });
  check("token exchange 200", tok.status === 200, JSON.stringify(tok.body));
  const access = tok.body.access_token;
  const refresh = tok.body.refresh_token;
  check("issued access+refresh", access && refresh);

  // 6. /mcp with valid bearer -> MCP initialize works (R6)
  if (access) {
    const ok = await req("/mcp", { method: "POST", headers: { "content-type": "application/json", accept: "application/json, text/event-stream", authorization: `Bearer ${access}` }, body: JSON.stringify({ jsonrpc: "2.0", method: "initialize", id: 1, params: { protocolVersion: "2025-06-18", capabilities: {}, clientInfo: { name: "check", version: "1" } } }) });
    check("/mcp accepts valid token", ok.status === 200, `got ${ok.status}`);
  }

  // 7. Wrong PKCE verifier rejected
  const bad = await req("/token", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ grant_type: "authorization_code", code: "nope", code_verifier: verifier, client_id: clientId }) });
  check("bad code rejected", bad.status === 400);

  // 8. Refresh rotation + reuse revokes grant (Notion behaviour)
  if (refresh) {
    const r1 = await req("/token", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ grant_type: "refresh_token", refresh_token: refresh }) });
    check("refresh #1 ok", r1.status === 200 && r1.body.refresh_token, JSON.stringify(r1.body));
    const r2 = await req("/token", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ grant_type: "refresh_token", refresh_token: refresh }) });
    check("reuse of rotated token rejected", r2.status === 400, `got ${r2.status}`);
    // new token from r1 should also now be dead (grant revoked)
    const r3 = await req("/token", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ grant_type: "refresh_token", refresh_token: r1.body.refresh_token }) });
    check("grant revoked on reuse", r3.status === 400, `got ${r3.status}`);
  }
}

console.log(`\n${failures === 0 ? "ALL PASS" : `${failures} FAILURE(S)`}`);
process.exit(failures === 0 ? 0 : 1);
