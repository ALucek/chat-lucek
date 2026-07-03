# Architecture

chat-lucek runs as two Cloud Run services behind a single domain. A Global HTTPS load balancer routes by path: API traffic goes to the Go service, everything else to the Next.js frontend. The API is the only service that reaches the database, the LLM provider, and secrets.

```mermaid
flowchart LR
    Browser -->|HTTPS| LB["Global HTTPS Load Balancer<br/>(Cloud Armor)"]
    LB -->|"/api/*, /readyz"| API["api<br/>(Go, Cloud Run)"]
    LB -->|everything else| WEB["web<br/>(Next.js, Cloud Run)"]
    API --> DB[("Cloud SQL<br/>Postgres")]
    API -->|SSE stream| LLM["OpenRouter"]
    API -.->|reads secrets| SM["Secret Manager"]
```

## Components

- **web** serves the Next.js App Router UI. It holds no data of its own; every dynamic action calls the API.
- **api** owns all state and integrations: authentication, conversation storage, and the streaming chat endpoint.
- **Cloud SQL** is the single Postgres instance backing the API.
- **OpenRouter** is the upstream LLM provider, consumed as a raw SSE stream.

## Streaming a reply

The API verifies ownership, persists the user message, streams tokens straight from OpenRouter to the browser, and saves the full reply once the stream closes.

```mermaid
sequenceDiagram
    participant B as Browser
    participant A as api
    participant P as Postgres
    participant O as OpenRouter
    B->>A: POST /api/conversations/{id}/messages
    A->>P: verify ownership, save user message
    A->>O: open streaming completion
    O-->>A: token deltas (SSE)
    A-->>B: token deltas (SSE)
    A->>P: save assistant message
```
