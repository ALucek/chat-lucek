import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test } from 'vitest';
import type { TreeNode } from '@/lib/run-log';
import { NodeRow } from './node-row';

const compactionNode: TreeNode = {
  id: 'r1:compaction',
  parent_id: null,
  type: 'compaction',
  text: 'a summary',
  children: [],
};

test('renders a top-level compaction node as a CompactionRow', () => {
  render(<NodeRow node={compactionNode} turnStreaming />);
  expect(screen.getByText('compacting')).toBeInTheDocument();
});

test('renders a compaction node nested inside a subagent drawer', async () => {
  const subagent: TreeNode = {
    id: 'sub1',
    parent_id: null,
    type: 'tool',
    name: 'run_subagent',
    input: { task: 'research the landscape' },
    children: [compactionNode],
  };
  render(<NodeRow node={subagent} turnStreaming />);
  // Subagent is collapsed, so the nested compaction is hidden.
  expect(screen.queryByText('compacting')).toBeNull();
  await userEvent.click(screen.getByRole('button'));
  expect(screen.getByText('compacting')).toBeInTheDocument();
});
