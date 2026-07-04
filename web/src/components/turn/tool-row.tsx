'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';
import { RowHeader } from './parts';

// stringify renders a tool value for the drill-down.
function stringify(v: unknown): string {
  if (v == null) return '';
  return typeof v === 'string' ? v : JSON.stringify(v, null, 2);
}

// ToolRow is a leaf tool: one line that expands to raw input/output.
export function ToolRow({
  node,
  active,
}: {
  node: TreeNode;
  active?: boolean;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="flex flex-col gap-1">
      <RowHeader
        label={toolLabel(node.name ?? '')}
        detail={inputDetail(node.input)}
        open={open}
        onToggle={() => setOpen((o) => !o)}
        active={active}
      />
      {open && (
        <div className="border-border bg-bg text-muted overflow-x-auto rounded-[var(--radius)] border p-2 text-xs">
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
