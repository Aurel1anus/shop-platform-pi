import { createServer } from 'node:http';
import { pathToFileURL } from 'node:url';

// Internal transport probe only: no model, tools, or business session is created.
export function createAgentServer() {
  return createServer((req, res) => {
    if (req.url?.split('?')[0] !== '/api/v1/agent/events') {
      res.writeHead(404).end();
      return;
    }
    if (req.method !== 'GET') {
      res.writeHead(405, { Allow: 'GET' }).end();
      return;
    }
    // This header is overwritten by Go after authentication. The service must
    // stay on a trusted internal network; this header is not a login credential.
    if (!req.headers['x-agent-account-id']) {
      res.writeHead(401).end();
      return;
    }
    res.writeHead(200, {
      'Content-Type': 'text/event-stream; charset=utf-8',
      'Cache-Control': 'no-cache',
      'X-Accel-Buffering': 'no',
    });
    const send = (event: string, data: object) => {
      res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
    };
    send('status', { status: 'connected', demo: true });
    let step = 0;
    const timer = setInterval(() => {
      if (step++ === 0) {
        send('text_delta', { text: 'SSE 转发链路已连接（固定测试文本）' });
      } else {
        send('done', { demo: true });
        res.end();
      }
    }, 300);
    res.on('close', () => clearInterval(timer));
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const port = Number(process.env.AGENT_PORT ?? 8082);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error('AGENT_PORT 必须是 1～65535 的整数');
  }
  const server = createAgentServer();
  server.on('error', (error) => {
    console.error(`Agent 服务启动失败：${error.message}`);
    process.exitCode = 1;
  });
  server.listen(port, '127.0.0.1', () => {
    console.log(`Agent SSE probe listening on http://127.0.0.1:${port}`);
  });
}
