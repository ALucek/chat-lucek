'use client';

import { useState } from 'react';
import type { TreeNode } from '@/lib/run-log';
import { Tag } from './parts';

// ReasoningRow is a dimmed one-line thought that expands to the full text.
export function ReasoningRow({ node }: { node: TreeNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="flex flex-col gap-0.5">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 text-left"
      >
        <Tag>thinking</Tag>
        {!open && (
          <span className="text-subtle min-w-0 flex-1 truncate text-xs">
            {node.text}
          </span>
        )}
        <span className="text-subtle ml-auto shrink-0 text-[11px]">
          {open ? '▾' : '▸'}
        </span>
      </button>
      {open && (
        <div className="text-subtle pl-0.5 text-xs whitespace-pre-wrap">
          {node.text}
        </div>
      )}
    </div>
  );
}
