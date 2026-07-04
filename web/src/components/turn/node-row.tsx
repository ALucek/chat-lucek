import type { TreeNode } from '@/lib/run-log';
import { ReasoningRow } from './reasoning-row';
import { ToolRow } from './tool-row';
import { SubagentRow } from './subagent-row';
import { TextBlock } from './text-block';

// NodeRow dispatches a node to its row (tool with children = subagent).
export function NodeRow({
  node,
  streaming,
  nested,
}: {
  node: TreeNode;
  streaming?: boolean;
  nested?: boolean;
}) {
  if (node.type === 'reasoning') return <ReasoningRow node={node} />;
  if (node.type === 'text')
    return <TextBlock node={node} nested={nested} streaming={streaming} />;
  if (node.children.length > 0)
    return (
      <SubagentRow
        node={node}
        renderChild={(c) => <NodeRow key={c.id} node={c} nested />}
      />
    );
  return <ToolRow node={node} />;
}
