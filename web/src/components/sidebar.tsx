'use client';

import { useRouter } from 'next/navigation';
import { useConversationsContext } from '@/lib/conversations-context';
import { ConversationItem } from './conversation-item';
import { Button } from '@/components/ui/button';
import { Skeleton } from './ui/skeleton';
import { UsageMeter } from './usage-meter';
import { SettingsMenu } from './settings-menu';

export function Sidebar() {
  const router = useRouter();
  const { conversations, loading, error, rename, remove } =
    useConversationsContext();

  return (
    <aside className="border-border bg-surface flex h-full w-64 flex-col border-r">
      <div className="border-border border-b p-3">
        <Button
          onClick={() => router.push('/')}
          className="w-full py-3 md:py-2"
        >
          New conversation
        </Button>
      </div>

      <nav className="flex-1 space-y-0.5 overflow-y-auto p-1.5">
        {loading && (
          <div className="space-y-1 p-1">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        )}
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

      <div className="border-border flex h-[var(--bottombar-h)] items-center gap-2 border-t px-3 text-sm">
        <SettingsMenu />
        <UsageMeter />
      </div>
    </aside>
  );
}
