import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SettingsMenu } from './settings-menu';
import { useAuth } from '@/lib/auth-context';

vi.mock('@/lib/auth-context');

function setDesktop(isDesktop: boolean) {
  window.matchMedia = vi.fn(
    (query: string) =>
      ({
        matches: isDesktop,
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList,
  );
}

const logout = vi.fn();

beforeEach(() => {
  vi.clearAllMocks();
  setDesktop(true);
  vi.mocked(useAuth).mockReturnValue({
    user: { id: 1, email: 'a@b.co' },
    status: 'authed',
    loginWithGoogle: vi.fn(),
    logout,
  } as unknown as ReturnType<typeof useAuth>);
});

afterEach(() => vi.restoreAllMocks());

describe('SettingsMenu', () => {
  it('opens the panel and shows the signed-in email', async () => {
    render(<SettingsMenu />);
    expect(screen.queryByText('a@b.co')).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Settings' }));
    expect(screen.getByText('a@b.co')).toBeInTheDocument();
  });

  it('logs out when Log out is clicked', async () => {
    render(<SettingsMenu />);
    await userEvent.click(screen.getByRole('button', { name: 'Settings' }));
    await userEvent.click(screen.getByRole('button', { name: 'Log out' }));
    expect(logout).toHaveBeenCalled();
  });
});
