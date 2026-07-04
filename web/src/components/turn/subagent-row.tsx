'use client';

import { useState } from 'react';
import type { ReactNode } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';
import { RowHeader, Block } from './parts';

// SubagentRow is a container tool: expands to a drawer with its task and its
// nested steps.
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
  const task = inputDetail(node.input);
  return (
    <div className="flex flex-col gap-1.5">
      <RowHeader
        label={toolLabel(node.name ?? '')}
        detail={task}
        open={open}
        onToggle={() => setOpen((o) => !o)}
        active={active}
        hasMore={node.children.length > 0 || task !== ''}
      />
      {open && (
        <Block>
          {task !== '' && (
            <div className="text-subtle text-xs break-words whitespace-pre-wrap">
              {task}
            </div>
          )}
          {node.children.map((c) => renderChild(c))}
        </Block>
      )}
    </div>
  );
}
