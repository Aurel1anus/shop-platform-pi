import assert from 'node:assert/strict';
import { once } from 'node:events';
import { test } from 'node:test';
import { createAgentServer } from '../src/server.ts';

test('Node probe streams fixed events and rejects unsupported requests', { timeout: 5000 }, async (t) => {
  const server = createAgentServer();
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  t.after(() => {
    server.closeAllConnections();
    server.close();
  });
  const origin = `http://127.0.0.1:${server.address().port}`;
  assert.equal((await fetch(`${origin}/api/v1/agent/events`)).status, 401);
  assert.equal((await fetch(`${origin}/api/v1/agent/sessions`)).status, 404);
  const wrongMethod = await fetch(`${origin}/api/v1/agent/events`, { method: 'POST' });
  assert.equal(wrongMethod.status, 405);
  assert.equal(wrongMethod.headers.get('allow'), 'GET');

  const response = await fetch(`${origin}/api/v1/agent/events?probe=1`, {
    headers: { 'X-Agent-Account-ID': 'test-account' },
  });
  assert.equal(response.status, 200);
  assert.match(response.headers.get('content-type'), /^text\/event-stream/);
  const reader = response.body.getReader();
  const first = new TextDecoder().decode((await reader.read()).value);
  assert.equal(first, 'event: status\ndata: {"status":"connected","demo":true}\n\n');
  let rest = '';
  const decoder = new TextDecoder();
  for (;;) {
    const { value, done } = await reader.read();
    if (done) {
      rest += decoder.decode();
      break;
    }
    rest += decoder.decode(value, { stream: true });
  }
  assert.equal(rest, 'event: text_delta\ndata: {"text":"SSE 转发链路已连接（固定测试文本）"}\n\nevent: done\ndata: {"demo":true}\n\n');
});
