'use client';

import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';

// Tag is the small uppercase label chip; it pulses while its step runs.
export function Tag({
  children,
  active,
}: {
  children: ReactNode;
  active?: boolean;
}) {
  return (
    <span
      className={`text-subtle shrink-0 rounded bg-black/5 px-1.5 pt-[3px] pb-[1px] text-[10px] tracking-[0.14em] uppercase ${active ? 'animate-pulse' : ''}`}
    >
      {children}
    </span>
  );
}

// Block is the height-capped, scrollable drawer a row expands into.
export function Block({ children }: { children: ReactNode }) {
  return (
    <div className="border-border bg-bg flex max-h-72 flex-col gap-2 overflow-y-auto rounded-[var(--radius)] border p-2.5">
      {children}
    </div>
  );
}

// RowHeader is the step line: chip, single-line detail, and a caret only when
// there is more to show (the detail overflows, or the row has a drawer).
export function RowHeader({
  label,
  detail,
  open,
  onToggle,
  active,
  hasMore,
}: {
  label: string;
  detail: string;
  open: boolean;
  onToggle: () => void;
  active?: boolean;
  hasMore?: boolean;
}) {
  const ref = useRef<HTMLSpanElement>(null);
  const [overflows, setOverflows] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (el) setOverflows(el.scrollWidth > el.clientWidth);
  }, [detail]);

  const expandable = overflows || !!hasMore;
  const chip = <Tag active={active}>{label}</Tag>;
  const text = (
    <span ref={ref} className="text-subtle min-w-0 flex-1 truncate text-xs">
      {detail}
    </span>
  );

  if (!expandable) {
    return (
      <div className="flex w-full items-baseline gap-2">
        {chip}
        {text}
      </div>
    );
  }
  return (
    <button
      type="button"
      aria-expanded={open}
      onClick={onToggle}
      className="flex w-full items-baseline gap-2 text-left"
    >
      {chip}
      {text}
      <span className="text-subtle ml-auto shrink-0 text-[11px]">
        {open ? '▾' : '▸'}
      </span>
    </button>
  );
}
