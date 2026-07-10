import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import MagicPage from './page';
import { useAuth } from '@/lib/auth-context';

const replace = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace }),
  useSearchParams: () => new URLSearchParams('token=tok-123'),
}));
vi.mock('@/lib/auth-context');

beforeEach(() => vi.resetAllMocks());

describe('MagicPage', () => {
  it('verifies the token then redirects home', async () => {
    const verifyMagicLink = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useAuth).mockReturnValue({
      verifyMagicLink,
    } as unknown as ReturnType<typeof useAuth>);
    render(<MagicPage />);
    await waitFor(() =>
      expect(verifyMagicLink).toHaveBeenCalledWith('tok-123'),
    );
    await waitFor(() => expect(replace).toHaveBeenCalledWith('/'));
  });

  it('shows an error when the link is invalid', async () => {
    const verifyMagicLink = vi.fn().mockRejectedValue(new Error('bad'));
    vi.mocked(useAuth).mockReturnValue({
      verifyMagicLink,
    } as unknown as ReturnType<typeof useAuth>);
    render(<MagicPage />);
    expect(await screen.findByRole('alert')).toBeInTheDocument();
  });
});
