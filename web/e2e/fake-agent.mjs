import { createServer } from 'node:http';

const PORT = process.env.FAKE_PORT ? Number(process.env.FAKE_PORT) : 8090;

const FRAMES = [
  'event: token\ndata: {"text":"Hello "}\n\n',
  'event: token\ndata: {"text":"from the stub."}\n\n',
  'event: usage\ndata: {"input":11,"output":7,"total":18,"reasoning":0}\n\n',
  'event: end\ndata: {}\n\n',
];

const server = createServer((req, res) => {
  // GET is Playwright's health probe; POST is the API's /run call.
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
  console.log(`fake-agent listening on :${PORT}`);
});
