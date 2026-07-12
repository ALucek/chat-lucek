import { Markdown } from '@/components/markdown';
import type { TreeNode } from '@/lib/run-log';

// TextBlock renders answer text: full-weight at top level, smaller when nested.
export function TextBlock({
  node,
  nested,
  streaming,
}: {
  node: TreeNode;
  nested?: boolean;
  streaming?: boolean;
}) {
  const text = node.text ?? '';
  if (nested) {
    return (
      <div className="markdown text-fg max-w-full min-w-0 text-[11px] break-words">
        <Markdown>{text}</Markdown>
      </div>
    );
  }
  return (
    <div className="markdown text-fg max-w-full min-w-0 text-sm break-words">
      <Markdown>{text}</Markdown>
      {streaming && (
        <span className="caret-blink" aria-hidden="true">
          ▍
        </span>
      )}
    </div>
  );
}
