'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter, useParams } from 'next/navigation';
import type { Conversation } from '@/lib/api';

interface Props {
  conversation: Conversation;
  rename: (id: number, title: string) => Promise<void>;
  remove: (id: number) => Promise<void>;
}

export function ConversationItem({ conversation, rename, remove }: Props) {
  const router = useRouter();
  const params = useParams();
  const isOpen = String(params.id) === String(conversation.id);

  const [editing, setEditing] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [draft, setDraft] = useState(conversation.title);

  function cancelEdit() {
    setDraft(conversation.title);
    setEditing(false);
  }

  async function saveRename() {
    const title = draft.trim();
    if (title && title !== conversation.title) {
      await rename(conversation.id, title);
    }
    setEditing(false);
  }

  async function confirmDelete() {
    await remove(conversation.id);
    if (isOpen) router.push('/');
  }

  if (editing) {
    return (
      <input
        autoFocus
        aria-label="Conversation title"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={cancelEdit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') saveRename();
          if (e.key === 'Escape') cancelEdit();
        }}
        className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
      />
    );
  }

  return (
    <div className="flex items-center gap-1 px-2 py-1">
      <Link
        href={`/c/${conversation.id}`}
        className={`flex-1 truncate text-sm ${isOpen ? 'font-semibold' : ''}`}
      >
        {conversation.title || 'New conversation'}
      </Link>
      {confirming ? (
        <span className="flex items-center gap-1 text-xs text-gray-600">
          Delete?
          <button onClick={confirmDelete} className="underline">
            yes
          </button>
          <button onClick={() => setConfirming(false)} className="underline">
            no
          </button>
        </span>
      ) : (
        <>
          <button
            onClick={() => setEditing(true)}
            aria-label="Rename"
            className="text-xs text-gray-500 hover:text-black"
          >
            Edit
          </button>
          <button
            onClick={() => setConfirming(true)}
            aria-label="Delete"
            className="text-xs text-gray-500 hover:text-black"
          >
            Delete
          </button>
        </>
      )}
    </div>
  );
}
