'use client';

import { useCallback, useState } from 'react';

const KEY = 'sidebar-collapsed';

export function useSidebarCollapsed() {
  const [collapsed, setCollapsed] = useState(
    () => typeof window !== 'undefined' && localStorage.getItem(KEY) === 'true',
  );
  const toggle = useCallback(() => {
    setCollapsed((c) => {
      const next = !c;
      localStorage.setItem(KEY, String(next));
      return next;
    });
  }, []);
  return { collapsed, toggle };
}
