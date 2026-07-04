import ReactMarkdown from 'react-markdown';
import { remarkPlugins, rehypePlugins } from '@/lib/markdown';
import type { TreeNode } from '@/lib/run-log';

function Markdown({ children }: { children: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={remarkPlugins}
      rehypePlugins={rehypePlugins}
      components={{
        a: (props) => (
          <a {...props} target="_blank" rel="noopener noreferrer" />
        ),
      }}
    >
      {children}
    </ReactMarkdown>
  );
}

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
