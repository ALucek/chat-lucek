'use client';

import { useRef, useState } from 'react';
import { sendFeedback, clearFeedback } from '@/lib/api';
import { useToast } from '@/lib/toast-context';
import { Dialog } from '@/components/ui/dialog';

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
    className="h-3.5 w-3.5"
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
    className="h-3.5 w-3.5"
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
    className="h-3.5 w-3.5"
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
    className="h-3.5 w-3.5"
  >
    <path d="M17 14V2" />
    <path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22a3.13 3.13 0 0 1-3-3.88Z" />
  </svg>
);

const iconBtn =
  'text-subtle hover:text-fg-strong hover:bg-hover flex h-7 w-7 items-center justify-center rounded-[var(--radius)]';

// MessageActions is the copy + thumbs row under an assistant reply.
export function MessageActions({
  messageId,
  content,
  initialRating = null,
  onRate,
}: {
  messageId: number;
  content: string;
  initialRating?: Rating | null;
  onRate?: (rating: Rating | null) => void;
}) {
  const [rating, setRating] = useState<Rating | null>(initialRating);
  const [note, setNote] = useState('');
  const [noteOpen, setNoteOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const { toast } = useToast();
  const taRef = useRef<HTMLTextAreaElement>(null);

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
    const prev = rating;
    if (next === rating) {
      // re-click the active thumb = clear the rating and its note
      setRating(null);
      setNote('');
      setNoteOpen(false);
      try {
        await clearFeedback(messageId);
        onRate?.(null); // clear the cached vote so it stays cleared on navigation
      } catch {
        setRating(prev);
        toast('Could not clear feedback');
      }
      return;
    }
    // Switching drops the old note (empty comment clears it from the trace).
    setRating(next);
    setNote('');
    setNoteOpen(true);
    try {
      await sendFeedback(messageId, next);
      onRate?.(next); // keep the cached vote in sync so it survives navigation
    } catch {
      setRating(prev);
      setNoteOpen(false);
      toast('Could not save feedback');
    }
  }

  async function submitNote() {
    const text = taRef.current?.value.trim() ?? '';
    setNoteOpen(false);
    if (rating === null || !text) return; // empty: the rating is already saved
    setNote(text);
    try {
      await sendFeedback(messageId, rating, text);
    } catch {
      toast('Could not save note');
    }
  }

  return (
    <div className="flex items-center gap-0.5">
      <button
        type="button"
        aria-label={copied ? 'Copied' : 'Copy response'}
        className={iconBtn}
        onClick={copy}
      >
        {copied ? <CheckIcon /> : <CopyIcon />}
      </button>
      <span className="bg-border mx-0.5 h-3.5 w-px" />
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

      <Dialog
        open={noteOpen}
        onClose={() => setNoteOpen(false)}
        label="Add a note"
      >
        <h2 className="text-fg-strong mb-3 flex items-center gap-1.5 text-sm">
          <span className="relative -top-px flex">
            {rating === -1 ? (
              <DownIcon filled={false} />
            ) : (
              <UpIcon filled={false} />
            )}
          </span>
          Add a note
        </h2>
        <textarea
          ref={taRef}
          defaultValue={note}
          rows={3}
          placeholder={rating === -1 ? 'What went wrong?' : 'What worked well?'}
          className="border-border bg-bg text-fg focus:border-fg-strong w-full resize-none rounded-[var(--radius)] border p-2.5 text-sm outline-none"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              submitNote();
            }
          }}
        />
        <p className="text-subtle mt-2 text-xs">
          Your note helps improve responses.
        </p>
        <div className="mt-4 flex justify-center gap-2 md:justify-end md:gap-1.5">
          <button
            type="button"
            onClick={() => setNoteOpen(false)}
            className="border-border text-muted hover:bg-hover flex-1 rounded-[var(--radius)] border px-3 py-2.5 text-sm md:flex-none md:py-1 md:text-xs"
          >
            Skip
          </button>
          <button
            type="button"
            onClick={submitNote}
            className="bg-accent text-accent-fg flex-1 rounded-[var(--radius)] px-3 py-2.5 text-sm md:flex-none md:py-1 md:text-xs"
          >
            Send
          </button>
        </div>
      </Dialog>
    </div>
  );
}
