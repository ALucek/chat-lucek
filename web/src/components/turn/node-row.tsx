import type { TreeNode } from '@/lib/run-log';
import { ReasoningRow } from './reasoning-row';
import { ToolRow } from './tool-row';
import { SubagentRow } from './subagent-row';
import { TextBlock } from './text-block';

// NodeRow dispatches a node to its row (tool with children = subagent). A tool
// with no output while the turn streams is still running.
export function NodeRow({
  node,
  turnStreaming,
  caret,
  nested,
}: {
  node: TreeNode;
  turnStreaming?: boolean;
  caret?: boolean;
  nested?: boolean;
}) {
  if (node.type === 'reasoning') return <ReasoningRow node={node} />;
  if (node.type === 'text') {
    // Skip blank text nodes; a whitespace-only node adds a phantom gap.
    if (!node.text?.trim()) return null;
    return <TextBlock node={node} nested={nested} streaming={caret} />;
  }
  const active = !!turnStreaming && node.output === undefined;
  if (node.children.length > 0)
    return (
      <SubagentRow
        node={node}
        active={active}
        renderChild={(c) => (
          <NodeRow key={c.id} node={c} turnStreaming={turnStreaming} nested />
        )}
      />
    );
  return <ToolRow node={node} active={active} />;
}
