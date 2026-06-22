import { createServer } from 'node:http';

const PORT = process.env.FAKE_PORT ? Number(process.env.FAKE_PORT) : 8090;

const FRAMES = [
  'data: {"choices":[{"delta":{"content":"Hello "}}]}\n\n',
  'data: {"choices":[{"delta":{"content":"from the stub."}}]}\n\n',
  'data: [DONE]\n\n',
];

const server = createServer((req, res) => {
  // GET is Playwright's health probe; POST is the API's chat-completions call.
  if (req.method !== 'POST') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('ok');
    return;
  }
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
  });
  for (const frame of FRAMES) res.write(frame);
  res.end();
});

server.listen(PORT, () => {
  console.log(`fake-openrouter listening on :${PORT}`);
});
