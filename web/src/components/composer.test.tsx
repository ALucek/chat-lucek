import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
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

  it('Escape blurs the box', () => {
    render(<Composer onSend={vi.fn()} onStop={vi.fn()} sending={false} />);
    const box = screen.getByRole('textbox');
    box.focus();
    expect(box).toHaveFocus();
    fireEvent.keyDown(box, { key: 'Escape' });
    expect(box).not.toHaveFocus();
  });

  it('shows Stop while sending and calls onStop', async () => {
    const onStop = vi.fn();
    render(<Composer onSend={vi.fn()} onStop={onStop} sending={true} />);
    expect(screen.getByRole('textbox')).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: /stop/i }));
    expect(onStop).toHaveBeenCalled();
  });

  it('captureTyping focuses the box when typing starts with nothing focused', () => {
    render(
      <Composer
        onSend={vi.fn()}
        onStop={vi.fn()}
        sending={false}
        captureTyping
      />,
    );
    const box = screen.getByRole('textbox');
    expect(box).not.toHaveFocus();
    fireEvent.keyDown(document.body, { key: 'h' });
    expect(box).toHaveFocus();
    expect(box).toHaveValue('h');
  });

  it('captureTyping ignores shortcut keys and non-printable keys', () => {
    render(
      <Composer
        onSend={vi.fn()}
        onStop={vi.fn()}
        sending={false}
        captureTyping
      />,
    );
    const box = screen.getByRole('textbox');
    fireEvent.keyDown(document.body, { key: 'k', metaKey: true });
    fireEvent.keyDown(document.body, { key: 'Tab' });
    expect(box).not.toHaveFocus();
  });

  it('does not capture typing when the prop is off', () => {
    render(<Composer onSend={vi.fn()} onStop={vi.fn()} sending={false} />);
    fireEvent.keyDown(document.body, { key: 'h' });
    expect(screen.getByRole('textbox')).not.toHaveFocus();
  });
});
