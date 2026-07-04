'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';

// stringify renders a tool value for the drill-down.
function stringify(v: unknown): string {
  if (v == null) return '';
  return typeof v === 'string' ? v : JSON.stringify(v, null, 2);
}

// ToolRow is a leaf tool: one line that expands to raw input/output.
export function ToolRow({ node }: { node: TreeNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="flex flex-col gap-0.5">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 text-left"
      >
        <span className="text-subtle shrink-0 text-[10px] tracking-[0.14em] uppercase">
          {toolLabel(node.name ?? '')}
        </span>
        <span className="text-subtle min-w-0 flex-1 truncate text-xs">
          {inputDetail(node.input)}
        </span>
        <span className="text-subtle ml-auto shrink-0 text-[11px]">
          {open ? '▾' : '▸'}
        </span>
      </button>
      {open && (
        <div className="border-border bg-bg text-muted mt-1 overflow-x-auto rounded-[var(--radius)] border p-2 text-xs">
          <div className="text-subtle text-[10px] tracking-[0.1em] uppercase">
            input
          </div>
          <pre className="break-words whitespace-pre-wrap">
            {stringify(node.input)}
          </pre>
          {node.output !== undefined && (
            <>
              <div className="text-subtle mt-2 text-[10px] tracking-[0.1em] uppercase">
                output
              </div>
              <pre className="break-words whitespace-pre-wrap">
                {stringify(node.output)}
              </pre>
            </>
          )}
        </div>
      )}
    </div>
  );
}
