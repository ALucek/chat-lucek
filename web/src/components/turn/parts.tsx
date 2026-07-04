import type { ReactNode } from 'react';

// Tag is the small uppercase label chip on the left of a timeline row.
export function Tag({ children }: { children: ReactNode }) {
  return (
    <span className="text-subtle shrink-0 rounded bg-black/5 px-1.5 py-0.5 text-[10px] tracking-[0.14em] uppercase">
      {children}
    </span>
  );
}

// Dot marks a step that is still running.
export function Dot() {
  return (
    <span
      aria-hidden="true"
      className="bg-fg-strong h-1.5 w-1.5 shrink-0 animate-pulse rounded-full"
    />
  );
}
