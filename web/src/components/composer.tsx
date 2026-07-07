'use client';

import { useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { autoSize } from '@/lib/autosize';

export function Composer({
  onSend,
  onStop,
  sending,
}: {
  onSend: (text: string) => void;
  onStop: () => void;
  sending: boolean;
}) {
  const [text, setText] = useState('');
  const [expanded, setExpanded] = useState(false);
  const ref = useRef<HTMLTextAreaElement>(null);

  function submit() {
    const trimmed = text.trim();
    if (!trimmed) return;
    onSend(trimmed);
    setText('');
    setExpanded(false);
    if (ref.current) ref.current.style.height = 'auto';
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  }

  return (
    <div className="border-border bg-surface flex items-end border-t px-3 pt-3 pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))]">
      <div
        className={`mx-auto flex w-full max-w-2xl gap-2 ${expanded ? 'items-end' : 'items-center'}`}
      >
        <Textarea
          ref={ref}
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            setExpanded(autoSize(e.target));
          }}
          onKeyDown={onKeyDown}
          disabled={sending}
          rows={1}
          placeholder="Send a message…"
          className="max-h-40 flex-1 overflow-x-hidden overflow-y-auto text-base sm:text-sm"
        />
        {sending ? (
          <Button type="button" variant="ghost" onClick={onStop}>
            Stop
          </Button>
        ) : (
          <Button type="button" onClick={submit}>
            Send
          </Button>
        )}
      </div>
    </div>
  );
}
