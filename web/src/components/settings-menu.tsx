'use client';

import { Menu } from '@/components/ui/menu';
import { useAuth } from '@/lib/auth-context';

export function SettingsMenu() {
  const { user, logout } = useAuth();

  return (
    <Menu
      label="Settings"
      placement="top-start"
      role="dialog"
      trigger={(p) => (
        // mirrors Button primary; raw <button> so Menu can attach its ref
        <button
          {...p}
          className="bg-accent text-accent-fg focus-visible:ring-accent flex-1 rounded-[var(--radius)] px-4 py-2 text-center text-sm font-medium transition-colors hover:opacity-90 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
        >
          Settings
        </button>
      )}
    >
      {({ close }) => (
        <div className="min-w-[216px]">
          <div className="border-border border-b px-2 py-2">
            <span className="text-subtle text-[11px] tracking-[0.18em] uppercase">
              Settings
            </span>
          </div>
          <div className="p-1">
            <p className="text-subtle px-2 pt-1 pb-1.5 text-[11px] tracking-[0.16em] uppercase">
              Account
            </p>
            <p className="text-fg-strong truncate px-2 pb-2 text-sm">
              {user?.email}
            </p>
            <button
              onClick={() => {
                close();
                logout();
              }}
              className="hover:bg-hover text-fg flex h-9 w-full items-center rounded px-2 text-sm md:h-8"
            >
              Log out
            </button>
          </div>
        </div>
      )}
    </Menu>
  );
}
