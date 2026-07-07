import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Dialog } from './dialog';

function Harness({ open }: { open: boolean }) {
  const onClose = vi.fn();
  return (
    <Dialog open={open} onClose={onClose} label="Test">
      <button>Inside</button>
    </Dialog>
  );
}

describe('Dialog', () => {
  it('renders its content only when open', () => {
    const { rerender } = render(<Harness open={false} />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    rerender(<Harness open={true} />);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Inside')).toBeInTheDocument();
  });

  it('closes on Escape', async () => {
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} label="Test">
        <button>Inside</button>
      </Dialog>,
    );
    await userEvent.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalled();
  });

  it('closes on backdrop click', async () => {
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} label="Test">
        <button>Inside</button>
      </Dialog>,
    );
    await userEvent.click(screen.getByTestId('dialog-backdrop'));
    expect(onClose).toHaveBeenCalled();
  });
});
