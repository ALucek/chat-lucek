'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';
import { Sidebar } from '@/components/sidebar';
import { Button } from '@/components/ui/button';
import { ConversationsProvider } from '@/lib/conversations-context';
import { UsageProvider } from '@/lib/usage-context';
import { MessagesProvider } from '@/lib/messages-context';
import { useSidebarCollapsed } from '@/lib/use-sidebar-collapsed';

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const router = useRouter();
  const { collapsed, toggle } = useSidebarCollapsed();

  useEffect(() => {
    if (status === 'anon') router.replace('/login');
  }, [status, router]);

  if (status !== 'authed') return null;
  return (
    <ConversationsProvider>
      <UsageProvider>
        <MessagesProvider>
          <div className="bg-bg relative flex h-screen">
            <Button
              variant="ghost"
              size="sm"
              onClick={toggle}
              aria-label="Toggle sidebar"
              aria-expanded={!collapsed}
              className="absolute top-3.5 left-3 z-20 h-8 w-8 p-0 text-lg leading-none"
            >
              ☰
            </Button>
            <div
              className={`overflow-hidden transition-[width] duration-200 ${
                collapsed ? 'w-0' : 'w-64'
              }`}
            >
              <Sidebar />
            </div>
            <main className="flex-1 overflow-hidden">{children}</main>
          </div>
        </MessagesProvider>
      </UsageProvider>
    </ConversationsProvider>
  );
}
