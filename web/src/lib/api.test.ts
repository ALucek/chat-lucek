import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  loginWithGoogle,
  me,
  refreshAccess,
  clearSession,
  ApiError,
  listConversations,
  createConversation,
  renameConversation,
  deleteConversation,
  getMessages,
  getUsage,
  parseSSE,
  sendMessage,
  setOnUnauthorized,
  exportAccount,
  deleteAccount,
} from './api';

// Minimal Response stand-in for a JSON body.
function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

beforeEach(() => {
  clearSession();
  vi.restoreAllMocks();
});

describe('api client', () => {
  it('loginWithGoogle stores the access token; later calls send the Bearer header', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(200, { access_token: 'a1' }))
      .mockResolvedValueOnce(jsonResponse(200, { id: 1, email: 'a@b.co' }));
    vi.stubGlobal('fetch', fetchMock);

    await loginWithGoogle('e2e:a@b.co');
    expect(fetchMock.mock.calls[0][0]).toBe('http://localhost:8080/api/google');
    expect((fetchMock.mock.calls[0][1] as RequestInit).credentials).toBe(
      'include',
    );

    await me();
    const headers = (fetchMock.mock.calls[1][1] as RequestInit)
      .headers as Record<string, string>;
    expect(headers['Authorization']).toBe('Bearer a1');
  });

  it('refreshes once and retries the request on a 401, sending no body', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' })) // me()
      .mockResolvedValueOnce(jsonResponse(200, { access_token: 'a2' })) // refresh
      .mockResolvedValueOnce(jsonResponse(200, { id: 1, email: 'a@b.co' })); // retry
    vi.stubGlobal('fetch', fetchMock);

    const user = await me();
    expect(user).toEqual({ id: 1, email: 'a@b.co' });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1][0]).toBe(
      'http://localhost:8080/api/refresh',
    );
    const refreshInit = fetchMock.mock.calls[1][1] as RequestInit;
    expect(refreshInit.credentials).toBe('include');
    expect(refreshInit.body).toBeUndefined();
  });

  it('shares a single refresh among concurrent callers', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse(200, { access_token: 'a2' }));
    vi.stubGlobal('fetch', fetchMock);

    const [t1, t2] = await Promise.all([refreshAccess(), refreshAccess()]);
    expect(t1).toBe('a2');
    expect(t2).toBe('a2');
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('clears the session and throws when refresh fails', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' })) // me()
      .mockResolvedValueOnce(
        jsonResponse(401, { error: 'invalid refresh token' }),
      ); // refresh
    vi.stubGlobal('fetch', fetchMock);

    await expect(me()).rejects.toBeInstanceOf(ApiError);
  });

  it('surfaces a friendly message on a 429', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(jsonResponse(429, {}));
    vi.stubGlobal('fetch', fetchMock);
    await expect(me()).rejects.toMatchObject({
      status: 429,
      message: 'Too many requests — please wait a moment and try again.',
    });
  });
});

describe('conversation endpoints', () => {
  it('GET wrappers request the right URL', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, []));
    vi.stubGlobal('fetch', fetchMock);
    await listConversations();
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/conversations',
    );
    await getMessages(7);
    expect(fetchMock.mock.calls[1][0]).toBe(
      'http://localhost:8080/api/conversations/7/messages',
    );
    await getUsage();
    expect(fetchMock.mock.calls[2][0]).toBe('http://localhost:8080/api/usage');
  });

  it('createConversation POSTs and returns the new conversation', async () => {
    const convo = { id: 5, title: '', created_at: 't', updated_at: 't' };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(201, convo));
    vi.stubGlobal('fetch', fetchMock);
    await expect(createConversation()).resolves.toEqual(convo);
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST' });
  });

  it('renameConversation PATCHes the title', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(204, null));
    vi.stubGlobal('fetch', fetchMock);
    await renameConversation(5, 'New name');
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('http://localhost:8080/api/conversations/5');
    expect(init).toMatchObject({ method: 'PATCH' });
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      title: 'New name',
    });
  });

  it('deleteConversation DELETEs', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(204, null));
    vi.stubGlobal('fetch', fetchMock);
    await deleteConversation(5);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('http://localhost:8080/api/conversations/5');
    expect(init).toMatchObject({ method: 'DELETE' });
  });
});

describe('parseSSE', () => {
  it('parses one complete frame', () => {
    const { events, rest } = parseSSE('event: delta\ndata: {"text":"hi"}\n\n');
    expect(events).toEqual([{ event: 'delta', data: '{"text":"hi"}' }]);
    expect(rest).toBe('');
  });

  it('parses multiple frames in one buffer', () => {
    const buf =
      'event: delta\ndata: {"text":"a"}\n\n' +
      'event: done\ndata: {"message_id":5}\n\n';
    const { events, rest } = parseSSE(buf);
    expect(events).toEqual([
      { event: 'delta', data: '{"text":"a"}' },
      { event: 'done', data: '{"message_id":5}' },
    ]);
    expect(rest).toBe('');
  });

  it('keeps a trailing partial frame in rest and completes it next call', () => {
    const first = parseSSE('event: delta\ndata: {"text":"hel');
    expect(first.events).toEqual([]);
    expect(first.rest).toBe('event: delta\ndata: {"text":"hel');
    const second = parseSSE(first.rest + 'lo"}\n\n');
    expect(second.events).toEqual([
      { event: 'delta', data: '{"text":"hello"}' },
    ]);
    expect(second.rest).toBe('');
  });

  it('returns no events for a blank or partial buffer', () => {
    expect(parseSSE('').events).toEqual([]);
    expect(parseSSE('event: delta').events).toEqual([]);
  });
});

function streamResponse(status: number, frames: string[]): Response {
  const encoder = new TextEncoder();
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const f of frames) controller.enqueue(encoder.encode(f));
      controller.close();
    },
  });
  return {
    ok: status >= 200 && status < 300,
    status,
    body,
    json: async () => ({}),
  } as Response;
}

describe('sendMessage', () => {
  it('dispatches node/delta/title and resolves on done', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        streamResponse(200, [
          'event: node\ndata: {"id":"a:text","parent_id":null,"type":"text"}\n\n',
          'event: delta\ndata: {"id":"a:text","text":"Hel"}\n\n',
          'event: delta\ndata: {"id":"a:text","text":"lo"}\n\n',
          'event: done\ndata: {"message_id":42}\n\n',
          'event: title\ndata: {"title":"Hi there"}\n\n',
        ]),
      );
    vi.stubGlobal('fetch', fetchMock);

    const nodeIds: string[] = [];
    const deltas: [string, string][] = [];
    let doneId = 0;
    let title = '';
    await sendMessage(7, 'hello', {
      onNode: (n) => nodeIds.push(n.id),
      onDelta: (id, t) => deltas.push([id, t]),
      onNodeEnd: () => {},
      onDone: (id) => {
        doneId = id;
      },
      onTitle: (t) => {
        title = t;
      },
      onError: () => {},
    });

    expect(nodeIds).toEqual(['a:text']);
    expect(deltas).toEqual([
      ['a:text', 'Hel'],
      ['a:text', 'lo'],
    ]);
    expect(doneId).toBe(42);
    expect(title).toBe('Hi there');
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('http://localhost:8080/api/conversations/7/messages');
    expect(init).toMatchObject({ method: 'POST' });
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      content: 'hello',
    });
  });

  it('calls onError on a non-ok initial response', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse(404, { error: 'conversation not found' }),
      );
    vi.stubGlobal('fetch', fetchMock);

    let err = '';
    await sendMessage(7, 'hello', {
      onNode: () => {},
      onDelta: () => {},
      onNodeEnd: () => {},
      onDone: () => {},
      onTitle: () => {},
      onError: (m) => {
        err = m;
      },
    });
    expect(err).toBe('conversation not found');
  });

  it('refreshes once and retries on a 401 initial response', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' })) // first POST
      .mockResolvedValueOnce(jsonResponse(200, { access_token: 'a2' })) // refresh
      .mockResolvedValueOnce(
        streamResponse(200, ['event: done\ndata: {"message_id":1}\n\n']),
      ); // retried POST
    vi.stubGlobal('fetch', fetchMock);

    let doneId = 0;
    await sendMessage(7, 'hello', {
      onNode: () => {},
      onDelta: () => {},
      onNodeEnd: () => {},
      onDone: (id) => {
        doneId = id;
      },
      onTitle: () => {},
      onError: () => {},
    });
    expect(doneId).toBe(1);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});

it('swallows an AbortError without calling onError', async () => {
  const fetchMock = vi
    .fn()
    .mockRejectedValue(new DOMException('aborted', 'AbortError'));
  vi.stubGlobal('fetch', fetchMock);

  const onError = vi.fn();
  const ac = new AbortController();
  await expect(
    sendMessage(
      7,
      'hi',
      {
        onNode: () => {},
        onDelta: () => {},
        onNodeEnd: () => {},
        onDone: () => {},
        onTitle: () => {},
        onError,
      },
      ac.signal,
    ),
  ).resolves.toBeUndefined();
  expect(onError).not.toHaveBeenCalled();
});

it('notifies onUnauthorized when a refresh fails', async () => {
  const cb = vi.fn();
  setOnUnauthorized(cb);
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' })) // me()
    .mockResolvedValueOnce(jsonResponse(401, { error: 'invalid refresh' })); // refresh
  vi.stubGlobal('fetch', fetchMock);

  await expect(me()).rejects.toBeInstanceOf(ApiError);
  expect(cb).toHaveBeenCalled();
  setOnUnauthorized(null);
});

describe('account data', () => {
  it('exportAccount refreshes and retries once on a 401', async () => {
    // exportAccount hand-rolls refresh/retry outside request(), so cover it here.
    const blob = new Blob(['{}'], { type: 'application/json' });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, status: 401 } as Response) // export → 401
      .mockResolvedValueOnce(jsonResponse(200, { access_token: 'a2' })) // refresh
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        blob: async () => blob,
      } as unknown as Response); // retry
    vi.stubGlobal('fetch', fetchMock);

    expect(await exportAccount()).toBe(blob);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1][0]).toBe(
      'http://localhost:8080/api/refresh',
    );
  });

  it('deleteAccount DELETEs the account endpoint', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue({ ok: true, status: 204 } as Response);
    vi.stubGlobal('fetch', fetchMock);
    await deleteAccount();
    const [url, opts] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/api/account');
    expect((opts as RequestInit).method).toBe('DELETE');
  });
});
