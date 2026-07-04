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
      className="bg-fg-strong h-1.5 w-1.5 shrink-0 animate-pulse self-center rounded-full"
    />
  );
}

// RowHeader is the shared step line: dot, tag, wrapping detail, and caret.
export function RowHeader({
  label,
  detail,
  open,
  onToggle,
  active,
}: {
  label: string;
  detail: string;
  open: boolean;
  onToggle: () => void;
  active?: boolean;
}) {
  return (
    <button
      type="button"
      aria-expanded={open}
      onClick={onToggle}
      className="flex w-full items-baseline gap-2 text-left"
    >
      {active && <Dot />}
      <Tag>{label}</Tag>
      <span
        className={`text-subtle min-w-0 flex-1 text-xs ${open ? 'break-words whitespace-pre-wrap' : 'truncate'}`}
      >
        {detail}
      </span>
      <span className="text-subtle ml-auto shrink-0 text-[11px]">
        {open ? '▾' : '▸'}
      </span>
    </button>
  );
}
