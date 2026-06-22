import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useMessages } from './use-messages';
import * as api from './api';
import { ApiError, type Message } from './api';

vi.mock('./api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api')>();
  return { ...actual, getMessages: vi.fn() };
});

const mA: Message[] = [{ id: 1, role: 'user', content: 'A', created_at: 't' }];
const mB: Message[] = [{ id: 2, role: 'user', content: 'B', created_at: 't' }];

beforeEach(() => {
  vi.mocked(api.getMessages).mockReset();
});

describe('useMessages', () => {
  it('loads messages for the id', async () => {
    vi.mocked(api.getMessages).mockResolvedValue(mA);
    const { result } = renderHook(({ id }) => useMessages(id), {
      initialProps: { id: 1 },
    });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.messages).toEqual(mA);
  });

  it('refetches when the id changes', async () => {
    vi.mocked(api.getMessages)
      .mockResolvedValueOnce(mA)
      .mockResolvedValueOnce(mB);
    const { result, rerender } = renderHook(({ id }) => useMessages(id), {
      initialProps: { id: 1 },
    });
    await waitFor(() => expect(result.current.messages).toEqual(mA));
    rerender({ id: 2 });
    await waitFor(() => expect(result.current.messages).toEqual(mB));
  });

  it('ignores a stale (out-of-order) response', async () => {
    let resolveSlow!: (v: Message[]) => void;
    const slow = new Promise<Message[]>((r) => {
      resolveSlow = r;
    });
    vi.mocked(api.getMessages)
      .mockReturnValueOnce(slow) // id=1, resolves late
      .mockResolvedValueOnce(mB); // id=2, resolves first
    const { result, rerender } = renderHook(({ id }) => useMessages(id), {
      initialProps: { id: 1 },
    });
    rerender({ id: 2 });
    await waitFor(() => expect(result.current.messages).toEqual(mB));
    await act(async () => {
      resolveSlow(mA); // stale id=1 response arrives now
    });
    expect(result.current.messages).toEqual(mB);
  });

  it('sets notFound on a 404', async () => {
    vi.mocked(api.getMessages).mockRejectedValue(
      new ApiError(404, 'conversation not found'),
    );
    const { result } = renderHook(({ id }) => useMessages(id), {
      initialProps: { id: 99 },
    });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.notFound).toBe(true);
  });
});
