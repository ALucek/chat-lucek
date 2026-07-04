'use client';

import { useState } from 'react';
import type { ReactNode } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';

// SubagentRow is a container tool: collapsed, expands to its subtree.
export function SubagentRow({
  node,
  renderChild,
}: {
  node: TreeNode;
  renderChild: (n: TreeNode) => ReactNode;
}) {
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
        <div className="border-border mt-1.5 ml-0.5 flex flex-col gap-2 border-l pl-3">
          {node.children.map((c) => renderChild(c))}
        </div>
      )}
    </div>
  );
}
