import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Composer } from './composer';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('Composer', () => {
  it('Enter submits the trimmed text and clears the box', async () => {
    const onSend = vi.fn();
    render(<Composer onSend={onSend} onStop={vi.fn()} sending={false} />);
    const box = screen.getByRole('textbox');
    await userEvent.type(box, 'hello{Enter}');
    expect(onSend).toHaveBeenCalledWith('hello');
    expect(box).toHaveValue('');
  });

  it('Shift+Enter inserts a newline and does not submit', async () => {
    const onSend = vi.fn();
    render(<Composer onSend={onSend} onStop={vi.fn()} sending={false} />);
    const box = screen.getByRole('textbox');
    await userEvent.type(box, 'a{Shift>}{Enter}{/Shift}b');
    expect(onSend).not.toHaveBeenCalled();
    expect(box).toHaveValue('a\nb');
  });

  it('does not submit empty or whitespace-only input', async () => {
    const onSend = vi.fn();
    render(<Composer onSend={onSend} onStop={vi.fn()} sending={false} />);
    const box = screen.getByRole('textbox');
    await userEvent.type(box, '   {Enter}');
    expect(onSend).not.toHaveBeenCalled();
  });

  it('shows Stop while sending and calls onStop', async () => {
    const onStop = vi.fn();
    render(<Composer onSend={vi.fn()} onStop={onStop} sending={true} />);
    expect(screen.getByRole('textbox')).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: /stop/i }));
    expect(onStop).toHaveBeenCalled();
  });

  function mockPointer(fine: boolean) {
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query === '(pointer: fine)' ? fine : false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })) as unknown as typeof window.matchMedia;
  }

  it('autoFocus focuses the box on a fine-pointer (desktop) device', () => {
    mockPointer(true);
    render(
      <Composer onSend={vi.fn()} onStop={vi.fn()} sending={false} autoFocus />,
    );
    expect(screen.getByRole('textbox')).toHaveFocus();
  });

  it('autoFocus does not focus on a coarse-pointer (touch) device', () => {
    mockPointer(false);
    render(
      <Composer onSend={vi.fn()} onStop={vi.fn()} sending={false} autoFocus />,
    );
    expect(screen.getByRole('textbox')).not.toHaveFocus();
  });
});
