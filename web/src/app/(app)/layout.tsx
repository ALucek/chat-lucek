'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';
import { Sidebar } from '@/components/sidebar';
import { ConversationsProvider } from '@/lib/conversations-context';
import { UsageProvider } from '@/lib/usage-context';
import { MessagesProvider } from '@/lib/messages-context';
import { useSidebarCollapsed } from '@/lib/use-sidebar-collapsed';
import { useMobileDrawer } from '@/lib/use-mobile-drawer';

// Sidebar-panel glyph (Lucide panel-left).
function SidebarIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className="h-[18px] w-[18px]"
    >
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M9 3v18" />
    </svg>
  );
}

// Edge handle riding the sidebar's right edge; caller sets position and size.
function SidebarHandle({
  expanded,
  onToggle,
  label,
  className,
}: {
  expanded: boolean;
  onToggle: () => void;
  label: string;
  className?: string;
}) {
  return (
    <button
      onClick={onToggle}
      aria-label={label}
      aria-expanded={expanded}
      className={`border-border bg-surface text-muted hover:text-fg-strong absolute top-3 items-center justify-center rounded-l-none rounded-r-[var(--radius)] border border-l-0 ${className ?? ''}`}
    >
      <SidebarIcon />
    </button>
  );
}

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const router = useRouter();
  const { collapsed, toggle } = useSidebarCollapsed();
  const { open, toggle: toggleMobile, close: closeMobile } = useMobileDrawer();

  useEffect(() => {
    if (status === 'anon') router.replace('/login');
  }, [status, router]);

  if (status !== 'authed') return null;
  return (
    <ConversationsProvider>
      <UsageProvider>
        <MessagesProvider>
          <div data-testid="app-shell" className="bg-bg relative flex h-dvh">
            {/* Desktop handle: collapses the push column; rides its right edge. */}
            <SidebarHandle
              expanded={!collapsed}
              onToggle={toggle}
              label="Toggle sidebar"
              className={`z-20 hidden h-9 w-6 transition-[left,color] duration-200 md:flex ${
                collapsed ? 'left-0' : 'left-64'
              }`}
            />
            {/* Backdrop: mobile only; fades in/out with the drawer. */}
            <div
              data-testid="backdrop"
              onClick={closeMobile}
              aria-hidden={!open}
              className={`fixed inset-x-0 top-0 z-30 h-dvh bg-black/40 transition-opacity duration-200 md:hidden ${
                open ? 'opacity-100' : 'pointer-events-none opacity-0'
              }`}
            />
            {/* Sidebar: fixed overlay on mobile, push column at md+. */}
            <div
              className={`fixed top-0 left-0 z-30 h-dvh w-64 transition-transform duration-200 md:static md:z-auto md:translate-x-0 md:overflow-hidden md:transition-[width] ${
                open ? 'translate-x-0' : '-translate-x-full'
              } ${collapsed ? 'md:w-0' : 'md:w-64'}`}
            >
              <Sidebar />
              {/* Mobile handle: inside the drawer so it rides its transform 1:1. */}
              <SidebarHandle
                expanded={open}
                onToggle={toggleMobile}
                label="Toggle menu"
                className="left-full z-40 flex h-11 w-8 transition-colors md:hidden"
              />
            </div>
            <main className="min-w-0 flex-1 overflow-hidden">{children}</main>
          </div>
        </MessagesProvider>
      </UsageProvider>
    </ConversationsProvider>
  );
}
