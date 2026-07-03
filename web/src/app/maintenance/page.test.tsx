import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import MaintenancePage from './page';

describe('MaintenancePage', () => {
  it('shows the message + legal links, with no sign-in button', () => {
    render(<MaintenancePage />);
    expect(screen.getByLabelText('Chat Łucek')).toBeInTheDocument();
    expect(screen.getByText('Down for maintenance.')).toBeInTheDocument();
    expect(screen.queryByTestId('google-signin')).toBeNull();
    expect(screen.getByRole('link', { name: 'Terms' })).toHaveAttribute(
      'href',
      '/terms',
    );
    expect(
      screen.getByRole('link', { name: 'Privacy Policy' }),
    ).toHaveAttribute('href', '/privacy');
  });
});
