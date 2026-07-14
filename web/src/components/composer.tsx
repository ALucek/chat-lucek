'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { autoSize } from '@/lib/autosize';

export function Composer({
  onSend,
  onStop,
  sending,
  captureTyping = false,
}: {
  onSend: (text: string) => void;
  onStop: () => void;
  sending: boolean;
  captureTyping?: boolean;
}) {
  const [text, setText] = useState('');
  const [expanded, setExpanded] = useState(false);
  const ref = useRef<HTMLTextAreaElement>(null);

  // Start typing anywhere (nothing else focused) and the keystroke lands here.
  useEffect(() => {
    if (!captureTyping) return;
    function focusOnType(e: KeyboardEvent) {
      const el = ref.current;
      if (!el || el.disabled) return;
      if (e.metaKey || e.ctrlKey || e.altKey || e.key.length !== 1) return;
      const active = document.activeElement;
      if (active && active !== document.body) return;
      el.focus();
    }
    document.addEventListener('keydown', focusOnType);
    return () => document.removeEventListener('keydown', focusOnType);
  }, [captureTyping]);

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
