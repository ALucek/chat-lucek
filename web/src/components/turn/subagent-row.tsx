'use client';

import { useState } from 'react';
import type { ReactNode } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';
import { Tag, Dot } from './parts';

// SubagentRow is a container tool: collapsed, expands to a bounded subtree.
export function SubagentRow({
  node,
  active,
  renderChild,
}: {
  node: TreeNode;
  active?: boolean;
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
        {active && <Dot />}
        <Tag>{toolLabel(node.name ?? '')}</Tag>
        <span
          className={`text-subtle min-w-0 flex-1 text-xs ${open ? 'break-words whitespace-normal' : 'truncate'}`}
        >
          {inputDetail(node.input)}
        </span>
        <span className="text-subtle ml-auto shrink-0 text-[11px]">
          {open ? '▾' : '▸'}
        </span>
      </button>
      {open && (
        <div className="border-border mt-1.5 ml-0.5 flex max-h-72 flex-col gap-2 overflow-y-auto border-l pr-1 pl-3">
          {node.children.map((c) => renderChild(c))}
        </div>
      )}
    </div>
  );
}
