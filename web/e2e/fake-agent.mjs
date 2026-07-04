import { createServer } from 'node:http';

const PORT = process.env.FAKE_PORT ? Number(process.env.FAKE_PORT) : 8090;

const FRAMES = [
  'event: node\ndata: {"id":"r1:reasoning","parent_id":null,"type":"reasoning"}\n\n',
  'event: delta\ndata: {"id":"r1:reasoning","text":"Planning the search."}\n\n',
  'event: node\ndata: {"id":"SA","parent_id":null,"type":"tool","name":"run_subagent","input":{"task":"research the topic"}}\n\n',
  'event: node\ndata: {"id":"s1","parent_id":"SA","type":"tool","name":"internet_search","input":{"query":"stub query"}}\n\n',
  'event: node_end\ndata: {"id":"s1","output":{"results":["a","b"]}}\n\n',
  'event: node\ndata: {"id":"m1:text","parent_id":"SA","type":"text"}\n\n',
  'event: delta\ndata: {"id":"m1:text","text":"subagent summary"}\n\n',
  'event: node_end\ndata: {"id":"SA","output":"done"}\n\n',
  'event: node\ndata: {"id":"a:text","parent_id":null,"type":"text"}\n\n',
  'event: delta\ndata: {"id":"a:text","text":"Hello from the stub."}\n\n',
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
