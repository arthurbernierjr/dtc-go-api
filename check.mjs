#!/usr/bin/env node
// Checks a deployed Todos API against the shared contract used by every backend
// side quest and every frontend guide in Deeper Than Code.
//
//   node check.mjs https://your-api.up.railway.app
//
// Needs Node 18 or later. No packages. Prints one line per check and exits 1 on
// the first batch of failures, so a green run means the frontends will work.

const base = (process.argv[2] || '').replace(/\/+$/, '');
if (!base) {
  console.error('usage: node check.mjs <base-url>');
  process.exit(2);
}

const ORIGIN = 'https://example.test';
const results = [];
let created = null;

function record(ok, name, detail) {
  results.push({ ok, name, detail });
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${ok || !detail ? '' : `  (${detail})`}`);
}

function keysEqual(obj, expected) {
  if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return false;
  const keys = Object.keys(obj).sort();
  return JSON.stringify(keys) === JSON.stringify([...expected].sort());
}

async function call(method, path, body, extraHeaders = {}) {
  const headers = { Origin: ORIGIN, ...extraHeaders };
  let payload;
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }
  const res = await fetch(base + path, { method, headers, body: payload });
  const text = await res.text();
  let json = null;
  if (text) {
    try { json = JSON.parse(text); } catch { json = undefined; }
  }
  return { status: res.status, headers: res.headers, text, json };
}

function hasCors(r) {
  const allow = r.headers.get('access-control-allow-origin');
  return allow === '*' || allow === ORIGIN;
}

async function check(name, fn) {
  try {
    const detail = await fn();
    record(detail === true || detail === undefined, name, detail === true || detail === undefined ? '' : String(detail));
  } catch (err) {
    record(false, name, err.message);
  }
}

// 1. health
await check('GET /health answers 200 {"ok":true}', async () => {
  const r = await call('GET', '/health');
  if (r.status !== 200) return `status ${r.status}`;
  if (!keysEqual(r.json, ['ok']) || r.json.ok !== true) return `body ${r.text.slice(0, 80)}`;
  return true;
});

// 2. CORS preflights
await check('OPTIONS /todos preflight allows POST with Content-Type from any origin', async () => {
  const r = await call('OPTIONS', '/todos', undefined, {
    'Access-Control-Request-Method': 'POST',
    'Access-Control-Request-Headers': 'content-type',
  });
  if (r.status !== 200 && r.status !== 204) return `status ${r.status}`;
  if (!hasCors(r)) return `allow-origin ${r.headers.get('access-control-allow-origin')}`;
  const methods = (r.headers.get('access-control-allow-methods') || '').toUpperCase();
  if (!methods.includes('POST') && !methods.includes('*')) return `allow-methods ${methods || 'missing'}`;
  const hdrs = (r.headers.get('access-control-allow-headers') || '').toLowerCase();
  if (!hdrs.includes('content-type') && !hdrs.includes('*')) return `allow-headers ${hdrs || 'missing'}`;
  return true;
});

await check('OPTIONS /todos/1 preflight allows DELETE', async () => {
  const r = await call('OPTIONS', '/todos/1', undefined, { 'Access-Control-Request-Method': 'DELETE' });
  if (r.status !== 200 && r.status !== 204) return `status ${r.status}`;
  if (!hasCors(r)) return `allow-origin ${r.headers.get('access-control-allow-origin')}`;
  const methods = (r.headers.get('access-control-allow-methods') || '').toUpperCase();
  if (!methods.includes('DELETE') && !methods.includes('*')) return `allow-methods ${methods || 'missing'}`;
  return true;
});

// 3. list
await check('GET /todos answers 200 with an array of {id,subject,details} ordered by id', async () => {
  const r = await call('GET', '/todos');
  if (r.status !== 200) return `status ${r.status}`;
  if (!hasCors(r)) return 'no access-control-allow-origin on the response';
  if (!Array.isArray(r.json)) return `body is not an array: ${r.text.slice(0, 80)}`;
  for (const item of r.json) {
    if (!keysEqual(item, ['id', 'subject', 'details'])) return `item keys ${Object.keys(item)}`;
    if (!Number.isInteger(item.id)) return `id ${item.id} is not an integer`;
  }
  for (let i = 1; i < r.json.length; i++) if (r.json[i].id <= r.json[i - 1].id) return 'ids are not ascending';
  return true;
});

// 4. validation
await check('POST /todos with an empty body answers 400 {"error":...}', async () => {
  const r = await call('POST', '/todos', {});
  if (r.status !== 400) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

await check('POST /todos with a blank subject answers 400', async () => {
  const r = await call('POST', '/todos', { subject: '   ', details: 'x' });
  if (r.status !== 400) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

await check('POST /todos without details answers 400', async () => {
  const r = await call('POST', '/todos', { subject: 'x' });
  if (r.status !== 400) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

// 5. create
await check('POST /todos answers 201 with exactly {id,subject,details}', async () => {
  const r = await call('POST', '/todos', { subject: 'check', details: 'created by check.mjs' });
  if (r.status !== 201) return `status ${r.status}`;
  if (!(r.headers.get('content-type') || '').includes('application/json')) return `content-type ${r.headers.get('content-type')}`;
  if (!keysEqual(r.json, ['id', 'subject', 'details'])) return `body ${r.text.slice(0, 80)}`;
  if (!Number.isInteger(r.json.id)) return `id ${r.json.id} is not an integer`;
  if (r.json.subject !== 'check' || r.json.details !== 'created by check.mjs') return 'values not echoed';
  created = r.json;
  return true;
});

// 6. show
await check('GET /todos/:id answers 200 with the same object', async () => {
  if (!created) return 'create failed';
  const r = await call('GET', `/todos/${created.id}`);
  if (r.status !== 200) return `status ${r.status}`;
  if (JSON.stringify(r.json) !== JSON.stringify(created)) return `body ${r.text.slice(0, 80)}`;
  return true;
});

await check('GET /todos/999999999 answers 404 {"error":"not found"}', async () => {
  const r = await call('GET', '/todos/999999999');
  if (r.status !== 404) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

// 7. update
await check('PUT /todos/:id answers 200 with the updated object', async () => {
  if (!created) return 'create failed';
  const r = await call('PUT', `/todos/${created.id}`, { subject: 'check two', details: 'updated by check.mjs' });
  if (r.status !== 200) return `status ${r.status}`;
  if (!keysEqual(r.json, ['id', 'subject', 'details'])) return `body ${r.text.slice(0, 80)}`;
  if (r.json.id !== created.id || r.json.subject !== 'check two' || r.json.details !== 'updated by check.mjs') return 'values not updated';
  return true;
});

await check('PUT /todos/:id with an empty body answers 400', async () => {
  if (!created) return 'create failed';
  const r = await call('PUT', `/todos/${created.id}`, {});
  if (r.status !== 400) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

await check('PUT /todos/999999999 answers 404', async () => {
  const r = await call('PUT', '/todos/999999999', { subject: 'x', details: 'y' });
  if (r.status !== 404) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

// 8. delete
await check('DELETE /todos/:id answers 204 with no body', async () => {
  if (!created) return 'create failed';
  const r = await call('DELETE', `/todos/${created.id}`);
  if (r.status !== 204) return `status ${r.status}`;
  if (r.text) return `body was not empty: ${r.text.slice(0, 40)}`;
  return true;
});

await check('DELETE /todos/:id a second time answers 404', async () => {
  if (!created) return 'create failed';
  const r = await call('DELETE', `/todos/${created.id}`);
  if (r.status !== 404) return `status ${r.status}`;
  if (!keysEqual(r.json, ['error'])) return `body ${r.text.slice(0, 80)}`;
  return true;
});

await check('GET /todos/:id after delete answers 404', async () => {
  if (!created) return 'create failed';
  const r = await call('GET', `/todos/${created.id}`);
  if (r.status !== 404) return `status ${r.status}`;
  return true;
});

const failed = results.filter((r) => !r.ok).length;
console.log(`\n${results.length - failed}/${results.length} checks passed for ${base}`);
process.exit(failed ? 1 : 0);
