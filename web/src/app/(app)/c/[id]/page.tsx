'use client';

import { useLayoutEffect, useRef } from 'react';
import { useParams } from 'next/navigation';
import { useStickToBottom } from 'use-stick-to-bottom';
import { useMessages } from '@/lib/messages-context';
import { MessageList } from '@/components/message-list';
import { Composer } from '@/components/composer';
import { Skeleton } from '@/components/ui/skeleton';

export default function ConversationPage() {
  const params = useParams();
  const id = Number(params.id);
  const {
    messages,
    loading,
    loadingOlder,
    hasMore,
    error,
    notFound,
    send,
    sending,
    loadOlder,
    stop,
  } = useMessages(id);
  const { scrollRef, contentRef } = useStickToBottom({ initial: 'instant' });

  // Scroll metrics captured when a load starts, restored after older messages prepend.
  const restore = useRef<{ height: number; top: number } | null>(null);

  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (el && restore.current) {
      el.scrollTop =
        el.scrollHeight - restore.current.height + restore.current.top;
      restore.current = null;
    }
  }, [messages, scrollRef]);

  // Fetch older messages near the top, anchoring the viewport so it doesn't jump.
  function onScroll(e: React.UIEvent<HTMLDivElement>) {
    const el = e.currentTarget;
    if (el.scrollTop < 200 && hasMore && !loadingOlder) {
      restore.current = { height: el.scrollHeight, top: el.scrollTop };
      loadOlder();
    }
  }

  if (notFound)
    return <p className="text-muted p-6 text-sm">Conversation not found</p>;

  return (
    <div className="flex h-full flex-col">
      {loading ? (
        <div className="flex-1 space-y-4 p-6">
          {[60, 40, 75].map((w, i) => (
            <Skeleton key={i} className="h-12" style={{ width: `${w}%` }} />
          ))}
        </div>
      ) : messages.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-1 text-center">
          <p className="text-fg-strong text-sm">No messages yet</p>
          <p className="text-muted text-sm">
            Send a message below to get started.
          </p>
        </div>
      ) : (
        <div
          ref={scrollRef}
          onScroll={onScroll}
          data-testid="messages-scroll"
          className="relative flex-1 overflow-y-auto [overflow-anchor:none]"
        >
          {loadingOlder && (
            <p className="text-subtle bg-surface/80 absolute inset-x-0 top-0 z-10 py-2 text-center text-xs">
              loading…
            </p>
          )}
          <div ref={contentRef}>
            <MessageList messages={messages} />
          </div>
        </div>
      )}
      {error && <p className="text-danger px-6 pb-4 text-sm">{error}</p>}
      <Composer onSend={send} onStop={stop} sending={sending} />
    </div>
  );
}
