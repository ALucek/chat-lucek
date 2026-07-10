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
      className={`group flex h-8 items-center gap-1.5 rounded-[var(--radius)] border-l-2 px-2 ${
        isOpen
          ? 'border-fg-strong bg-surface-muted'
          : 'hover:bg-hover border-transparent'
      }`}
    >
      <span
        className={`w-2.5 shrink-0 text-sm ${isOpen ? 'text-fg-strong' : 'text-subtle'}`}
      >
        {isOpen ? '>' : ''}
      </span>
      <Link
        href={`/c/${conversation.id}`}
        aria-current={isOpen ? 'page' : undefined}
        className={`flex-1 truncate text-sm ${isOpen ? 'text-fg-strong' : 'text-muted'}`}
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
            className="text-subtle hover:text-fg-strong flex h-5 w-5 shrink-0 items-center justify-center rounded text-base leading-none focus-visible:opacity-100 aria-expanded:opacity-100 md:opacity-0 md:group-hover:opacity-100"
          >
            ⋮
          </button>
        )}
      >
        {({ close }) => (
          <MenuContent
            title={conversation.title}
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
  title,
  onRename,
  onDelete,
}: {
  title: string;
  onRename: () => void;
  onDelete: () => Promise<boolean>;
}) {
  const [confirming, setConfirming] = useState(false);

  if (confirming) {
    return (
      <div className="p-1">
        <p className="text-fg-strong mb-2 text-center text-xs">
          Delete this conversation?
        </p>
        <div className="flex justify-center gap-1.5">
          <button
            onClick={() => setConfirming(false)}
            className="border-border text-muted hover:bg-hover rounded border px-3 py-1 text-xs"
          >
            Cancel
          </button>
          <button
            onClick={async () => {
              const ok = await onDelete();
              if (!ok) setConfirming(false);
            }}
            className="bg-danger rounded px-3 py-1 text-xs text-white hover:brightness-95"
          >
            Delete
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <div className="border-border border-b px-2 py-2 md:hidden">
        <span className="text-subtle block text-[11px] tracking-[0.16em] uppercase">
          Conversation
        </span>
        <span className="text-fg-strong block truncate text-sm">
          {title || 'New conversation'}
        </span>
      </div>
      <button
        onClick={onRename}
        className="hover:bg-hover text-fg flex h-9 items-center rounded px-2 text-sm md:h-8"
      >
        Rename
      </button>
      <button
        onClick={() => setConfirming(true)}
        className="hover:bg-hover text-danger flex h-9 items-center rounded px-2 text-sm md:h-8"
      >
        Delete
      </button>
    </div>
  );
}
