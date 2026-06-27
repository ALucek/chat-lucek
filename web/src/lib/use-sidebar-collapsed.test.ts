import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSidebarCollapsed } from './use-sidebar-collapsed';

beforeEach(() => {
  localStorage.clear();
});

describe('useSidebarCollapsed', () => {
  it('defaults to expanded when storage is empty', () => {
    const { result } = renderHook(() => useSidebarCollapsed());
    expect(result.current.collapsed).toBe(false);
  });

  it('initializes collapsed from storage', () => {
    localStorage.setItem('sidebar-collapsed', 'true');
    const { result } = renderHook(() => useSidebarCollapsed());
    expect(result.current.collapsed).toBe(true);
  });

  it('toggle flips the value and persists it', () => {
    const { result } = renderHook(() => useSidebarCollapsed());
    act(() => result.current.toggle());
    expect(result.current.collapsed).toBe(true);
    expect(localStorage.getItem('sidebar-collapsed')).toBe('true');
    act(() => result.current.toggle());
    expect(result.current.collapsed).toBe(false);
    expect(localStorage.getItem('sidebar-collapsed')).toBe('false');
  });
});
