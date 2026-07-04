'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { RowHeader, Block } from './parts';

// ReasoningRow is a dimmed one-line thought; expands to the full text.
export function ReasoningRow({ node }: { node: TreeNode }) {
  const [open, setOpen] = useState(false);
  const text = node.text ?? '';
  return (
    <div className="flex flex-col gap-1">
      <RowHeader
        label="thinking"
        detail={text}
        open={open}
        onToggle={() => setOpen((o) => !o)}
      />
      {open && (
        <Block>
          <div className="text-subtle text-xs break-words whitespace-pre-wrap">
            {text}
          </div>
        </Block>
      )}
    </div>
  );
}
