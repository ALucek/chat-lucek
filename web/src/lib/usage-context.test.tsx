import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { UsageProvider, useUsage } from './usage-context';
import * as api from './api';

vi.mock('./api');

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <UsageProvider>{children}</UsageProvider>
);

beforeEach(() => {
  vi.resetAllMocks();
});

describe('UsageProvider', () => {
  it('fetches usage on mount', async () => {
    vi.mocked(api.getUsage).mockResolvedValue({ used: 100, budget: 8192 });
    const { result } = renderHook(() => useUsage(), { wrapper });
    await waitFor(() => expect(result.current.used).toBe(100));
    expect(result.current.budget).toBe(8192);
  });

  it('refresh re-fetches the latest usage', async () => {
    vi.mocked(api.getUsage)
      .mockResolvedValueOnce({ used: 100, budget: 8192 })
      .mockResolvedValueOnce({ used: 250, budget: 8192 });
    const { result } = renderHook(() => useUsage(), { wrapper });
    await waitFor(() => expect(result.current.used).toBe(100));
    await act(async () => {
      result.current.refresh();
    });
    await waitFor(() => expect(result.current.used).toBe(250));
  });

  it('stays null when the fetch fails', async () => {
    vi.mocked(api.getUsage).mockRejectedValue(new Error('boom'));
    const { result } = renderHook(() => useUsage(), { wrapper });
    await waitFor(() => expect(api.getUsage).toHaveBeenCalled());
    expect(result.current.used).toBeNull();
    expect(result.current.budget).toBeNull();
  });
});
