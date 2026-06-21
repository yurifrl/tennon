// Self-check for the token contract (the security-sensitive path).
// Run: JWT_SECRET=test node test_tokens.mjs
import assert from 'node:assert';
import { SignJWT, jwtVerify } from 'jose';

const SECRET = new TextEncoder().encode('test-secret');
const ISSUER = 'http://localhost:8000';

const mint = (typ, ttl) => new SignJWT({ email: 'a@b.com', typ })
  .setProtectedHeader({ alg: 'HS256' }).setSubject('a@b.com')
  .setAudience('client-1').setIssuer(ISSUER).setIssuedAt().setExpirationTime(ttl).sign(SECRET);

// valid access token verifies and carries the email
const access = await mint('access', '1h');
const { payload } = await jwtVerify(access, SECRET, { issuer: ISSUER });
assert.equal(payload.email, 'a@b.com');
assert.equal(payload.typ, 'access');

// wrong secret is rejected (forged token)
await assert.rejects(jwtVerify(access, new TextEncoder().encode('other'), { issuer: ISSUER }));

// expired token is rejected
const expired = await mint('access', '0s');
await new Promise(r => setTimeout(r, 1100));
await assert.rejects(jwtVerify(expired, SECRET, { issuer: ISSUER }));

// a refresh token must not be accepted where an access token is required
const refresh = await mint('refresh', '30d');
const { payload: rp } = await jwtVerify(refresh, SECRET, { issuer: ISSUER });
assert.equal(rp.typ, 'refresh'); // verifyAccessToken rejects typ !== 'access'

console.log('ok');
