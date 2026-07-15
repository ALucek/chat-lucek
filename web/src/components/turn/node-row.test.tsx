import { render } from '@testing-library/react';
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

test('compaction nodes render nothing (inert seam)', () => {
  const { container } = render(<NodeRow node={compactionNode} />);
  expect(container.firstChild).toBeNull();
});
