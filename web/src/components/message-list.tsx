import ReactMarkdown from 'react-markdown';
import { remarkPlugins, rehypePlugins } from '@/lib/markdown';
import type { ChatMessage } from '@/lib/messages-context';
import { buildTree } from '@/lib/run-log';
import { NodeRow } from './turn/node-row';
import { MessageActions } from './message-actions';

// AssistantMessage renders the inline run timeline, or plain content when a
// reply has no node log (older or trivial replies).
function AssistantMessage({ m }: { m: ChatMessage }) {
  if (m.nodes && m.nodes.length > 0) {
    const tree = buildTree(m.nodes);
    // Caret marks the live edge: only when the run's last node is text.
    const last = tree[tree.length - 1];
    const caretId = last?.type === 'text' ? last.id : undefined;
    return (
      <div className="border-border bg-surface-muted flex w-full flex-col gap-2.5 rounded-[var(--radius)] border px-4 py-3">
        {tree.map((n) => (
          <NodeRow
            key={n.id}
            node={n}
            turnStreaming={m.streaming}
            caret={m.streaming && n.id === caretId}
          />
        ))}
      </div>
    );
  }
  return (
    <div className="markdown text-fg max-w-full min-w-0 text-sm break-words">
      <ReactMarkdown
        remarkPlugins={remarkPlugins}
        rehypePlugins={rehypePlugins}
        components={{
          a: (props) => (
            <a {...props} target="_blank" rel="noopener noreferrer" />
          ),
        }}
      >
        {m.content}
      </ReactMarkdown>
      {m.streaming && (
        <span className="caret-blink" aria-hidden="true">
          ▍
        </span>
      )}
    </div>
  );
}

export function MessageList({ messages }: { messages: ChatMessage[] }) {
  return (
    <ul
      aria-live="polite"
      className="mx-auto flex max-w-2xl flex-col gap-4 px-4 py-5 sm:px-5 sm:py-7"
    >
      {messages.map((m) => {
        const isUser = m.role === 'user';
        return (
          <li
            key={m.id}
            className={`flex flex-col gap-1.5 ${isUser ? 'items-end' : 'items-start'}`}
          >
            {isUser ? (
              <span className="border-border bg-surface-muted text-fg max-w-[80%] min-w-0 rounded-[var(--radius)] border px-3 py-2 text-sm break-words whitespace-pre-wrap">
                {m.content}
              </span>
            ) : (
              <>
                <AssistantMessage m={m} />
                {!m.streaming && (
                  <MessageActions
                    messageId={m.id}
                    content={m.content}
                    initialRating={m.feedback?.rating ?? null}
                  />
                )}
              </>
            )}
          </li>
        );
      })}
    </ul>
  );
}
