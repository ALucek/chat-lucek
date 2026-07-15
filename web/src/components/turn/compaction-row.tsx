'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { RowHeader, Block } from './parts';

// CompactionRow marks where older history was summarized. It pulses while the
// summarizer streams; expand to read the folded summary.
export function CompactionRow({
  node,
  active,
}: {
  node: TreeNode;
  active?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const text = node.text ?? '';
  return (
    <div className="flex flex-col gap-1.5">
      <RowHeader
        label="compacting"
        detail="Summarizing earlier conversation…"
        open={open}
        onToggle={() => setOpen((o) => !o)}
        active={active}
        hasMore={text !== ''}
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
