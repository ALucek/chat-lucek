'use client';

import { useCallback, useState } from 'react';
import { usePathname } from 'next/navigation';

export function useMobileDrawer() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const [prevPathname, setPrevPathname] = useState(pathname);

  // Close the drawer whenever navigation changes the route. Adjusting state
  // during render (rather than in an effect) is React's recommended pattern for
  // resetting state in response to a changed value.
  if (pathname !== prevPathname) {
    setPrevPathname(pathname);
    setOpen(false);
  }

  const toggle = useCallback(() => setOpen((o) => !o), []);
  const close = useCallback(() => setOpen(false), []);

  return { open, toggle, close };
}
