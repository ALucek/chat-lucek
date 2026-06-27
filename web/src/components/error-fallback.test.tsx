import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ErrorFallback } from './error-fallback';

describe('ErrorFallback', () => {
  it('shows a message and calls onReset on Try again', async () => {
    const onReset = vi.fn();
    render(<ErrorFallback onReset={onReset} />);
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(onReset).toHaveBeenCalled();
  });
});
