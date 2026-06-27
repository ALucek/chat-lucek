import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { ToastProvider, useToast } from './toast-context';

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('useToast', () => {
  it('adds a toast and auto-dismisses after the TTL', () => {
    const { result } = renderHook(() => useToast(), { wrapper: ToastProvider });
    act(() => result.current.toast('save failed'));
    expect(result.current.toasts).toHaveLength(1);
    expect(result.current.toasts[0].message).toBe('save failed');
    act(() => vi.advanceTimersByTime(4000));
    expect(result.current.toasts).toHaveLength(0);
  });

  it('dismisses manually', () => {
    const { result } = renderHook(() => useToast(), { wrapper: ToastProvider });
    act(() => result.current.toast('oops'));
    const id = result.current.toasts[0].id;
    act(() => result.current.dismiss(id));
    expect(result.current.toasts).toHaveLength(0);
  });
});
