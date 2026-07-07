import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DeleteAccountDialog } from './delete-account-dialog';
import { useAuth } from '@/lib/auth-context';
import { deleteAccount } from '@/lib/api';
import { ToastProvider } from '@/lib/toast-context';

vi.mock('@/lib/auth-context');
vi.mock('@/lib/api');

const push = vi.fn();
vi.mock('next/navigation', () => ({ useRouter: () => ({ push }) }));

const logout = vi.fn();
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <ToastProvider>{children}</ToastProvider>
);

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(useAuth).mockReturnValue({
    user: { id: 1, email: 'a@b.co' },
    status: 'authed',
    loginWithGoogle: vi.fn(),
    logout,
  } as unknown as ReturnType<typeof useAuth>);
});

describe('DeleteAccountDialog', () => {
  it('arms Delete only when the typed email matches', async () => {
    render(<DeleteAccountDialog open onClose={vi.fn()} />, { wrapper });
    const del = screen.getByRole('button', { name: 'Delete account' });
    expect(del).toBeDisabled();
    await userEvent.type(screen.getByLabelText('Confirm email'), '  A@B.CO  ');
    expect(del).toBeEnabled();
  });

  it('deletes, logs out, and redirects on confirm', async () => {
    vi.mocked(deleteAccount).mockResolvedValue();
    render(<DeleteAccountDialog open onClose={vi.fn()} />, { wrapper });
    await userEvent.type(screen.getByLabelText('Confirm email'), 'a@b.co');
    await userEvent.click(
      screen.getByRole('button', { name: 'Delete account' }),
    );
    expect(deleteAccount).toHaveBeenCalled();
    expect(logout).toHaveBeenCalled();
    expect(push).toHaveBeenCalledWith('/login');
  });
});
