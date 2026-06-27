import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useMobileDrawer } from './use-mobile-drawer';

let pathname = '/c/1';
vi.mock('next/navigation', () => ({
  usePathname: () => pathname,
}));

beforeEach(() => {
  pathname = '/c/1';
});

describe('useMobileDrawer', () => {
  it('defaults to closed', () => {
    const { result } = renderHook(() => useMobileDrawer());
    expect(result.current.open).toBe(false);
  });

  it('toggle flips the open state', () => {
    const { result } = renderHook(() => useMobileDrawer());
    act(() => result.current.toggle());
    expect(result.current.open).toBe(true);
    act(() => result.current.toggle());
    expect(result.current.open).toBe(false);
  });

  it('close sets open to false', () => {
    const { result } = renderHook(() => useMobileDrawer());
    act(() => result.current.toggle());
    expect(result.current.open).toBe(true);
    act(() => result.current.close());
    expect(result.current.open).toBe(false);
  });

  it('closes when the pathname changes', () => {
    const { result, rerender } = renderHook(() => useMobileDrawer());
    act(() => result.current.toggle());
    expect(result.current.open).toBe(true);
    pathname = '/c/2';
    rerender();
    expect(result.current.open).toBe(false);
  });
});
