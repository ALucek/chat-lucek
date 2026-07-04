'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { RowHeader } from './parts';

// ReasoningRow is a dimmed one-line thought that wraps to the full text.
export function ReasoningRow({ node }: { node: TreeNode }) {
  const [open, setOpen] = useState(false);
  return (
    <RowHeader
      label="thinking"
      detail={node.text ?? ''}
      open={open}
      onToggle={() => setOpen((o) => !o)}
    />
  );
}
