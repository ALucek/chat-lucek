# Architecture

chat-lucek runs as three Cloud Run services. A Global HTTPS load balancer fronts two of them, routing by path: API traffic to the Go service, everything else to the Next.js frontend. The third, the agent, sits behind the API: it grants the invoke role only to the API's service account, which calls it with an ID token. The API owns the database and secrets; the agent owns the LLM provider and web search.

```mermaid
flowchart LR
    Browser -->|HTTPS| LB["Global HTTPS Load Balancer<br/>(Cloud Armor)"]
    LB -->|"/api/*, /readyz"| API["api<br/>(Go, Cloud Run)"]
    LB -->|everything else| WEB["web<br/>(Next.js, Cloud Run)"]
    API --> DB[("Cloud SQL<br/>Postgres")]
    API -->|SSE run stream| AGENT["agent<br/>(Python, Cloud Run)"]
    AGENT -->|completions| LLM["OpenRouter"]
    AGENT -->|web search| TAVILY["Tavily"]
    AGENT -.->|traces| LS["LangSmith"]
    API -.->|reads secrets| SM["Secret Manager"]
```

## Components

- **web** serves the Next.js App Router UI. It holds no data of its own; every dynamic action calls the API.
- **api** owns all state and auth: sign-in (Google or an email magic link), conversation storage, and the chat endpoint, which runs the agent and relays its stream to the browser.
- **agent** is a LangGraph agent behind one streaming `/run` endpoint. It runs the model loop and emits its run as an ordered event stream.
- **Cloud SQL** is the single Postgres instance backing the API.
- **OpenRouter** is the upstream LLM provider, and **Tavily** backs the agent's web search.
- **LangSmith** receives a trace of every agent run, for step-level debugging of reasoning, tools, and subagents.

## Streaming a reply

The API verifies ownership, persists the user message, opens a run on the agent, relays the agent's event stream to the browser, and saves the reply and its run trace once the stream closes.

```mermaid
sequenceDiagram
    participant B as Browser
    participant A as api
    participant P as Postgres
    participant G as agent
    participant O as OpenRouter
    participant L as LangSmith
    B->>A: POST /api/conversations/{id}/messages
    A->>P: verify ownership, save user message
    A->>G: POST /run (message history)
    G->>O: run the agent loop
    G--)L: trace the run
    G-->>A: run events (SSE)
    A-->>B: run events (SSE)
    A->>P: save assistant message + trace
```
