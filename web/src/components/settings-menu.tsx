'use client';

import { useState } from 'react';
import { Menu } from '@/components/ui/menu';
import { DeleteAccountDialog } from '@/components/delete-account-dialog';
import { useAuth } from '@/lib/auth-context';
import { useToast } from '@/lib/toast-context';
import { exportAccount } from '@/lib/api';

export function SettingsMenu() {
  const { user, logout } = useAuth();
  const { toast } = useToast();
  const [deleteOpen, setDeleteOpen] = useState(false);

  async function exportData() {
    try {
      const blob = await exportAccount();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `chat-lucek-export-${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      toast('Could not export your data');
    }
  }

  const rowClass =
    'hover:bg-hover text-fg flex h-11 w-full items-center rounded px-2 text-sm md:h-8';
  const labelClass =
    'text-subtle px-2 pt-1 pb-1.5 text-[11px] tracking-[0.16em] uppercase';

  return (
    <>
      <Menu
        label="Settings"
        placement="top-start"
        role="dialog"
        trigger={(p) => (
          // mirrors Button primary; raw <button> so Menu can attach its ref
          <button
            {...p}
            className="bg-accent text-accent-fg focus-visible:ring-accent flex-1 rounded-[var(--radius)] px-4 py-3 text-center text-sm font-medium transition-colors hover:opacity-90 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none md:py-2"
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
              <p className={labelClass}>Account</p>
              <p className="text-fg-strong truncate px-2 pb-2 text-sm">
                {user?.email}
              </p>
              <button
                onClick={() => {
                  close();
                  logout();
                }}
                className={rowClass}
              >
                Log out
              </button>
            </div>
            <div className="border-border border-t p-1">
              <p className={labelClass}>Data</p>
              <button
                onClick={() => {
                  close();
                  exportData();
                }}
                className={rowClass}
              >
                Export data
              </button>
              <button
                onClick={() => {
                  close();
                  setDeleteOpen(true);
                }}
                className={`${rowClass} !text-danger`}
              >
                Delete account
              </button>
            </div>
          </div>
        )}
      </Menu>
      <DeleteAccountDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
      />
    </>
  );
}
