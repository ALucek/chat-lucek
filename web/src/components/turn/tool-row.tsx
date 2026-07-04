'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { toolLabel, inputDetail } from '@/lib/run-log';
import { RowHeader, Block } from './parts';

// stringify renders a tool value for the drawer.
function stringify(v: unknown): string {
  if (v == null) return '';
  return typeof v === 'string' ? v : JSON.stringify(v, null, 2);
}

// ToolRow is a leaf tool: the header carries the input digest, so the drawer
// shows the result (falling back to the full input while it runs).
export function ToolRow({
  node,
  active,
}: {
  node: TreeNode;
  active?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const output = stringify(node.output);
  const body = output || stringify(node.input);
  return (
    <div className="flex flex-col gap-1">
      <RowHeader
        label={toolLabel(node.name ?? '')}
        detail={inputDetail(node.input)}
        open={open}
        onToggle={() => setOpen((o) => !o)}
        active={active}
        hasMore={output !== ''}
      />
      {open && (
        <Block>
          <pre className="text-muted text-xs break-words whitespace-pre-wrap">
            {body}
          </pre>
        </Block>
      )}
    </div>
  );
}
