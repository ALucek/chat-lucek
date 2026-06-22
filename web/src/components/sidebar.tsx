'use client';

import { useRouter } from 'next/navigation';
import { useConversationsContext } from '@/lib/conversations-context';
import { useAuth } from '@/lib/auth-context';
import { ConversationItem } from './conversation-item';
import { Button } from '@/components/ui/button';

export function Sidebar() {
  const router = useRouter();
  const { user, logout } = useAuth();
  const { conversations, loading, error, create, rename, remove } =
    useConversationsContext();

  async function onNew() {
    const convo = await create();
    router.push(`/c/${convo.id}`);
  }

  return (
    <aside className="border-border bg-surface flex h-full w-64 flex-col border-r">
      <div className="border-border border-b p-2">
        <Button onClick={onNew} className="w-full">
          New conversation
        </Button>
      </div>

      <nav className="flex-1 overflow-y-auto p-1">
        {loading && <p className="text-muted p-2 text-sm">Loading…</p>}
        {error && <p className="text-danger p-2 text-sm">{error}</p>}
        {!loading &&
          !error &&
          conversations.map((c) => (
            <ConversationItem
              key={c.id}
              conversation={c}
              rename={rename}
              remove={remove}
            />
          ))}
      </nav>

      <div className="border-border border-t p-3 text-sm">
        <p className="text-muted truncate">{user?.email}</p>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logout()}
          className="mt-1 px-0"
        >
          Log out
        </Button>
      </div>
    </aside>
  );
}
