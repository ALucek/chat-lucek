'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter, useParams } from 'next/navigation';
import type { Conversation } from '@/lib/api';
import { Input } from '@/components/ui/input';
import { Menu } from '@/components/ui/menu';
import { useToast } from '@/lib/toast-context';

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
  const [draft, setDraft] = useState(conversation.title);
  const { toast } = useToast();

  function cancelEdit() {
    setDraft(conversation.title);
    setEditing(false);
  }

  async function saveRename() {
    const title = draft.trim();
    if (title && title !== conversation.title) {
      try {
        await rename(conversation.id, title);
      } catch {
        toast('Could not rename conversation');
        cancelEdit();
        return;
      }
    }
    setEditing(false);
  }

  async function confirmDelete(): Promise<boolean> {
    try {
      await remove(conversation.id);
    } catch {
      toast('Could not delete conversation');
      return false;
    }
    if (isOpen) router.push('/');
    return true;
  }

  if (editing) {
    return (
      <Input
        autoFocus
        aria-label="Conversation title"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={cancelEdit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') saveRename();
          if (e.key === 'Escape') cancelEdit();
        }}
        className="px-2 py-1"
      />
    );
  }

  return (
    <div
      className={`group flex h-12 items-center gap-1.5 rounded-[var(--radius)] px-2 md:h-8 ${
        isOpen ? 'bg-black/[0.04]' : 'hover:bg-hover'
      }`}
    >
      <span
        className={`w-2.5 shrink-0 text-base md:text-sm ${isOpen ? 'text-fg-strong' : 'text-subtle'}`}
      >
        {isOpen ? '>' : ''}
      </span>
      <Link
        href={`/c/${conversation.id}`}
        aria-current={isOpen ? 'page' : undefined}
        className={`flex-1 truncate text-base md:text-sm ${isOpen ? 'text-fg-strong' : 'text-muted'}`}
      >
        {conversation.title || 'New conversation'}
      </Link>
      <Menu
        label="Conversation actions"
        placement="bottom-start"
        trigger={(p) => (
          <button
            {...p}
            aria-label="Conversation actions"
            className="text-subtle hover:text-fg-strong flex h-11 w-11 shrink-0 items-center justify-center rounded text-xl leading-none focus-visible:opacity-100 aria-expanded:opacity-100 md:h-5 md:w-5 md:text-base md:opacity-0 md:group-hover:opacity-100"
          >
            ⋮
          </button>
        )}
      >
        {({ close }) => (
          <MenuContent
            onRename={() => {
              close();
              setEditing(true);
            }}
            onDelete={confirmDelete}
          />
        )}
      </Menu>
    </div>
  );
}

function MenuContent({
  onRename,
  onDelete,
}: {
  onRename: () => void;
  onDelete: () => Promise<boolean>;
}) {
  const [confirming, setConfirming] = useState(false);

  if (confirming) {
    return (
      <div className="p-1">
        <p className="text-fg-strong mb-3 text-center text-sm md:mb-2 md:text-xs">
          Delete this conversation?
        </p>
        <div className="flex justify-center gap-2 md:gap-1.5">
          <button
            onClick={() => setConfirming(false)}
            className="border-border text-muted hover:bg-hover flex-1 rounded border px-3 py-2.5 text-sm md:flex-none md:py-1 md:text-xs"
          >
            Cancel
          </button>
          <button
            onClick={async () => {
              const ok = await onDelete();
              if (!ok) setConfirming(false);
            }}
            className="bg-danger flex-1 rounded px-3 py-2.5 text-sm text-white hover:brightness-95 md:flex-none md:py-1 md:text-xs"
          >
            Delete
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <button
        onClick={onRename}
        className="hover:bg-hover text-fg flex h-12 items-center rounded px-2 text-base md:h-8 md:text-sm"
      >
        Rename
      </button>
      <button
        onClick={() => setConfirming(true)}
        className="hover:bg-hover text-danger flex h-12 items-center rounded px-2 text-base md:h-8 md:text-sm"
      >
        Delete
      </button>
    </div>
  );
}
