'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Dialog } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { useAuth } from '@/lib/auth-context';
import { useToast } from '@/lib/toast-context';
import { deleteAccount } from '@/lib/api';

export function DeleteAccountDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { user, logout } = useAuth();
  const router = useRouter();
  const { toast } = useToast();
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);

  const matches =
    !!user?.email && confirm.trim().toLowerCase() === user.email.toLowerCase();

  async function onDelete() {
    setBusy(true);
    try {
      await deleteAccount(confirm.trim());
    } catch {
      setBusy(false);
      toast('Could not delete your account');
      return;
    }
    await logout();
    toast('Account deleted');
    router.push('/login');
  }

  return (
    <Dialog open={open} onClose={onClose} label="Delete account">
      <h2 className="text-fg-strong mb-2 text-sm">Delete account</h2>
      <p className="text-muted mb-3 text-sm">
        This permanently deletes your account and all conversations. This cannot
        be undone. Type your email to confirm.
      </p>
      <Input
        aria-label="Confirm email"
        value={confirm}
        onChange={(e) => setConfirm(e.target.value)}
        placeholder={user?.email}
        className="mb-3 px-2 py-1"
      />
      <div className="flex justify-end gap-1.5">
        <button
          onClick={onClose}
          className="border-border text-muted hover:bg-hover rounded border px-3 py-1 text-xs"
        >
          Cancel
        </button>
        <button
          onClick={onDelete}
          disabled={!matches || busy}
          className="bg-danger rounded px-3 py-1 text-xs text-white hover:brightness-95 disabled:opacity-50"
        >
          Delete account
        </button>
      </div>
    </Dialog>
  );
}
