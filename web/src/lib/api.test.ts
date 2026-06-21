import { describe, it, expect, vi, beforeEach } from 'vitest';
import { login, me, refreshAccess, clearSession, ApiError } from './api';

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
  it('login stores tokens and later calls send the Bearer header', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse(200, { access_token: 'a1', refresh_token: 'r1' }),
      )
      .mockResolvedValueOnce(jsonResponse(200, { id: 1, email: 'a@b.co' }));
    vi.stubGlobal('fetch', fetchMock);

    await login('a@b.co', 'password123');
    expect(localStorage.getItem('refresh_token')).toBe('r1');

    await me();
    const headers = (fetchMock.mock.calls[1][1] as RequestInit)
      .headers as Record<string, string>;
    expect(headers['Authorization']).toBe('Bearer a1');
  });

  it('refreshes once and retries the request on a 401', async () => {
    localStorage.setItem('refresh_token', 'r1');
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' })) // me()
      .mockResolvedValueOnce(jsonResponse(200, { access_token: 'a2' })) // refresh
      .mockResolvedValueOnce(jsonResponse(200, { id: 1, email: 'a@b.co' })); // retry
    vi.stubGlobal('fetch', fetchMock);

    const user = await me();
    expect(user).toEqual({ id: 1, email: 'a@b.co' });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('shares a single refresh among concurrent callers', async () => {
    localStorage.setItem('refresh_token', 'r1');
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
    localStorage.setItem('refresh_token', 'r1');
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' })) // me()
      .mockResolvedValueOnce(
        jsonResponse(401, { error: 'invalid refresh token' }),
      ); // refresh
    vi.stubGlobal('fetch', fetchMock);

    await expect(me()).rejects.toBeInstanceOf(ApiError);
    expect(localStorage.getItem('refresh_token')).toBeNull();
  });
});
