'use client';

import { useState } from 'react';
import type { ReactNode } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';
import { RowHeader } from './parts';

// SubagentRow is a container tool: collapsed, expands to a boxed subtree.
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
    <div className="flex flex-col gap-1">
      <RowHeader
        label={toolLabel(node.name ?? '')}
        detail={inputDetail(node.input)}
        open={open}
        onToggle={() => setOpen((o) => !o)}
        active={active}
      />
      {open && (
        <div className="border-border bg-bg flex max-h-72 flex-col gap-2 overflow-y-auto rounded-[var(--radius)] border p-2.5">
          {node.children.map((c) => renderChild(c))}
        </div>
      )}
    </div>
  );
}
