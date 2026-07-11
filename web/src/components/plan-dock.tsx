'use client';

import { useEffect, useRef, useState } from 'react';
import type { RunNode } from '@/lib/api';
import { latestPlan, type PlanProgress } from '@/lib/run-log';

const GLYPH: Record<PlanProgress, string> = {
  completed: '●',
  'in progress': '◐',
  outstanding: '○',
};

const GLYPH_CLASS: Record<PlanProgress, string> = {
  completed: 'text-subtle',
  'in progress': 'text-fg-strong animate-pulse motion-reduce:animate-none',
  outstanding: 'text-subtle opacity-60',
};

const TEXT_CLASS: Record<PlanProgress, string> = {
  completed: 'text-subtle line-through decoration-black/30',
  'in progress': 'text-fg-strong',
  outstanding: 'text-subtle',
};

// PlanDock shows the latest set_todos plan above the composer.
export function PlanDock({
  nodes,
  running,
}: {
  nodes: RunNode[];
  running: boolean;
}) {
  const todos = latestPlan(nodes);

  // Expanded while running, collapsed when idle; manual toggle overrides.
  const [override, setOverride] = useState<boolean | null>(null);
  const prevRunning = useRef(running);
  useEffect(() => {
    if (prevRunning.current !== running) {
      setOverride(null);
      prevRunning.current = running;
    }
  }, [running]);

  if (!todos) return null;

  const open = override ?? running;
  const done = todos.filter((t) => t.progress === 'completed').length;
  const complete = done === todos.length;
  const doing = todos.find((t) => t.progress === 'in progress');
  const now = complete ? 'Plan complete' : (doing?.description ?? 'Planning…');

  return (
    <div className="px-3">
      <div className="mx-auto w-full max-w-2xl pb-2">
        <div className="border-border bg-surface overflow-hidden rounded-[var(--radius)] border">
          <button
            type="button"
            aria-expanded={open}
            onClick={() => setOverride(!open)}
            className="hover:bg-hover flex w-full items-center gap-2.5 px-3 py-2.5 text-left"
          >
            <span
              className={`text-subtle shrink-0 rounded bg-black/5 px-1.5 pt-[3px] pb-[1px] text-[10px] tracking-[0.14em] uppercase ${running ? 'animate-pulse motion-reduce:animate-none' : ''}`}
            >
              {complete ? 'plan ✓' : 'plan'}
            </span>
            <span className="text-muted min-w-0 flex-1 truncate text-xs">
              {now}
            </span>
            <span className="hidden h-1 w-16 shrink-0 overflow-hidden rounded-full bg-black/10 sm:block">
              <span
                className="bg-fg-strong block h-full rounded-full transition-[width] duration-300"
                style={{ width: `${(done / todos.length) * 100}%` }}
              />
            </span>
            <span className="text-subtle shrink-0 text-[11px] tabular-nums">
              {done}/{todos.length}
            </span>
            <span className="text-subtle shrink-0 text-[11px]">
              {open ? '▾' : '▸'}
            </span>
          </button>
          {open && (
            <ul className="border-border flex flex-col gap-0.5 border-t px-3 py-2.5">
              {todos.map((t, i) => (
                <li
                  key={i}
                  className="flex items-baseline gap-2.5 py-0.5 text-[13px]"
                >
                  <span
                    className={`w-4 shrink-0 text-center text-xs ${GLYPH_CLASS[t.progress]}`}
                  >
                    {GLYPH[t.progress]}
                  </span>
                  <span className={TEXT_CLASS[t.progress]}>
                    {t.description}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
