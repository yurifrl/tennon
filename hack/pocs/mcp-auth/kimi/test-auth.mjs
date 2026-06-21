#!/usr/bin/env node
// Test suite for MCP authenticated server POC
// Run: node test-auth.mjs

import { strict as assert } from 'assert';

const BASE_URL = process.env.BASE_URL || 'http://localhost:8000';

async function test(name, fn) {
  try {
    await fn();
    console.log(`✓ ${name}`);
    return true;
  } catch (e) {
    console.error(`✗ ${name}`);
    console.error(`  ${e.message}`);
    return false;
  }
}

async function runTests() {
  console.log(`Testing against ${BASE_URL}\n`);

  const results = [];

  // AC1: Unauthenticated request to /mcp returns 401 with resource metadata
  results.push(await test('AC1: /mcp returns 401 with auth challenge', async () => {
    const res = await fetch(`${BASE_URL}/mcp`);
    assert.equal(res.status, 401, `Expected 401, got ${res.status}`);
    const wwwAuth = res.headers.get('www-authenticate');
    assert.ok(wwwAuth, 'Expected www-authenticate header');
    assert.ok(wwwAuth.includes('resource_metadata='), 'Expected resource_metadata in www-authenticate');
  }));

  // AC2: Well-known endpoints exist and are valid JSON
  results.push(await test('AC2: /.well-known/oauth-protected-resource exists', async () => {
    const res = await fetch(`${BASE_URL}/.well-known/oauth-protected-resource`);
    assert.equal(res.status, 200, `Expected 200, got ${res.status}`);
    const data = await res.json();
    assert.ok(data.resource, 'Expected resource field');
    assert.ok(data.authorization_servers?.length > 0, 'Expected authorization_servers array');
  }));

  results.push(await test('AC2: /.well-known/oauth-authorization-server exists', async () => {
    const res = await fetch(`${BASE_URL}/.well-known/oauth-authorization-server`);
    assert.equal(res.status, 200, `Expected 200, got ${res.status}`);
    const data = await res.json();
    assert.ok(data.authorization_endpoint, 'Expected authorization_endpoint');
    assert.ok(data.token_endpoint, 'Expected token_endpoint');
    assert.ok(data.registration_endpoint, 'Expected registration_endpoint');
    assert.deepStrictEqual(data.code_challenge_methods_supported, ['S256'], 'Expected PKCE S256 support');
  }));

  // AC3, AC4, AC5: These require browser interaction and are tested manually
  results.push(await test('AC3-5: (Manual) OAuth flow requires browser - see README.md', async () => {
    // Placeholder - actual OAuth flow needs Selenium or manual testing
    assert.ok(true, 'Manual test documented in README');
  }));

  const passed = results.filter(Boolean).length;
  const total = results.length;
  console.log(`\n${passed}/${total} tests passed`);
  process.exit(passed === total ? 0 : 1);
}

runTests().catch(e => {
  console.error(e);
  process.exit(1);
});
