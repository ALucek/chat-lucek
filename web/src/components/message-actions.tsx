'use client';

import { useEffect, useRef, useState } from 'react';
import { sendFeedback } from '@/lib/api';
import { useToast } from '@/lib/toast-context';

type Rating = -1 | 1;

const CopyIcon = () => (
  <svg
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={2}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
    className="h-4 w-4"
  >
    <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
    <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
  </svg>
);

const CheckIcon = () => (
  <svg
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={2}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
    className="h-4 w-4"
  >
    <path d="M20 6 9 17l-5-5" />
  </svg>
);

const UpIcon = ({ filled }: { filled: boolean }) => (
  <svg
    viewBox="0 0 24 24"
    fill={filled ? 'currentColor' : 'none'}
    stroke="currentColor"
    strokeWidth={2}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
    className="h-4 w-4"
  >
    <path d="M7 10v12" />
    <path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z" />
  </svg>
);

const DownIcon = ({ filled }: { filled: boolean }) => (
  <svg
    viewBox="0 0 24 24"
    fill={filled ? 'currentColor' : 'none'}
    stroke="currentColor"
    strokeWidth={2}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
    className="h-4 w-4"
  >
    <path d="M17 14V2" />
    <path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22a3.13 3.13 0 0 1-3-3.88Z" />
  </svg>
);

const iconBtn =
  'text-subtle hover:text-fg-strong hover:bg-hover flex h-8 w-8 items-center justify-center rounded-[var(--radius)]';

// MessageActions is the copy + binding thumbs row under an assistant reply.
export function MessageActions({
  messageId,
  content,
}: {
  messageId: number;
  content: string;
}) {
  const [rating, setRating] = useState<Rating | null>(null);
  const [note, setNote] = useState('');
  const [noteOpen, setNoteOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const { toast } = useToast();
  const barRef = useRef<HTMLDivElement>(null);
  const taRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (!noteOpen) return;
    const onDown = (e: MouseEvent) => {
      if (barRef.current && !barRef.current.contains(e.target as Node)) {
        setNoteOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setNoteOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [noteOpen]);

  useEffect(() => {
    const ta = taRef.current;
    if (noteOpen && ta) {
      ta.style.height = 'auto';
      ta.style.height = `${ta.scrollHeight}px`;
      ta.focus();
    }
  }, [noteOpen]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast('Could not copy');
    }
  }

  async function vote(next: Rating) {
    if (next === rating) {
      setNoteOpen(true); // re-click the active thumb = edit the note
      return;
    }
    const prev = rating;
    setRating(next);
    setNoteOpen(true);
    try {
      await sendFeedback(messageId, next, note || undefined);
    } catch {
      setRating(prev);
      setNoteOpen(false);
      toast('Could not save feedback');
    }
  }

  async function submitNote() {
    if (rating === null) return;
    const text = taRef.current?.value.trim() ?? '';
    setNoteOpen(false);
    if (!text) return; // empty: the rating is already saved, nothing to attach
    setNote(text);
    try {
      await sendFeedback(messageId, rating, text);
    } catch {
      toast('Could not save note');
    }
  }

  return (
    <div ref={barRef} className="relative mt-1 flex items-center gap-0.5">
      <button
        type="button"
        aria-label={copied ? 'Copied' : 'Copy response'}
        className={iconBtn}
        onClick={copy}
      >
        {copied ? <CheckIcon /> : <CopyIcon />}
      </button>
      <span className="bg-border mx-1 h-4 w-px" />
      <button
        type="button"
        aria-label="Good response"
        aria-pressed={rating === 1}
        className={`${iconBtn} ${rating === 1 ? 'text-fg-strong' : ''}`}
        onClick={() => vote(1)}
      >
        <UpIcon filled={rating === 1} />
      </button>
      <button
        type="button"
        aria-label="Bad response"
        aria-pressed={rating === -1}
        className={`${iconBtn} ${rating === -1 ? 'text-fg-strong' : ''}`}
        onClick={() => vote(-1)}
      >
        <DownIcon filled={rating === -1} />
      </button>

      {noteOpen && (
        <div className="border-border bg-surface absolute top-full left-0 z-10 mt-1.5 w-80 max-w-[calc(100vw-2rem)] rounded-[var(--radius)] border p-2.5 shadow-lg">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-subtle text-[10px] tracking-[0.14em] uppercase">
              Add a note
            </span>
            <button
              type="button"
              aria-label="Dismiss note"
              className="text-subtle hover:text-fg-strong rounded px-1 text-xs"
              onClick={() => setNoteOpen(false)}
            >
              ✕
            </button>
          </div>
          <textarea
            ref={taRef}
            defaultValue={note}
            rows={2}
            placeholder="optional"
            className="border-border bg-bg text-fg max-h-36 w-full resize-none rounded-[var(--radius)] border p-2 text-sm outline-none"
            onInput={(e) => {
              const el = e.currentTarget;
              el.style.height = 'auto';
              el.style.height = `${el.scrollHeight}px`;
            }}
          />
          <div className="mt-2 flex justify-end">
            <button
              type="button"
              className="bg-accent text-accent-fg rounded-[var(--radius)] px-3 py-1 text-xs"
              onClick={submitNote}
            >
              Send
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
