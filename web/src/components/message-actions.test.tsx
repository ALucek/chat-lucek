import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MessageActions } from './message-actions';
import { sendFeedback } from '@/lib/api';

const toast = vi.fn();
vi.mock('@/lib/api', () => ({
  sendFeedback: vi.fn().mockResolvedValue(undefined),
  clearFeedback: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('@/lib/toast-context', () => ({ useToast: () => ({ toast }) }));

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MessageActions', () => {
  it('renders copy and two thumb buttons', () => {
    render(<MessageActions messageId={1} content="hello" />);
    expect(screen.getByRole('button', { name: /copy/i })).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /good response/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /bad response/i }),
    ).toBeInTheDocument();
  });

  it('copies the raw content', async () => {
    const user = userEvent.setup();
    // Define after setup() so this spy wins over userEvent's clipboard stub.
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    });
    render(<MessageActions messageId={1} content="**md** body" />);
    await user.click(screen.getByRole('button', { name: /copy/i }));
    expect(writeText).toHaveBeenCalledWith('**md** body');
  });

  it('records a thumb up and opens the note editor', async () => {
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" />);
    await user.click(screen.getByRole('button', { name: /good response/i }));
    expect(sendFeedback).toHaveBeenCalledWith(7, 1, undefined);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('switches to thumb down', async () => {
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" />);
    await user.click(screen.getByRole('button', { name: /good response/i }));
    await user.click(screen.getByRole('button', { name: /bad response/i }));
    expect(sendFeedback).toHaveBeenLastCalledWith(7, -1, undefined);
  });

  it('clears feedback when the active thumb is re-clicked', async () => {
    const { clearFeedback } = await import('@/lib/api');
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" initialRating={1} />);
    const up = screen.getByRole('button', { name: /good response/i });
    expect(up).toHaveAttribute('aria-pressed', 'true');
    await user.click(up);
    expect(clearFeedback).toHaveBeenCalledWith(7);
    expect(up).toHaveAttribute('aria-pressed', 'false');
  });

  it('does not open the note dialog when the active thumb is re-clicked', async () => {
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" initialRating={1} />);
    await user.click(screen.getByRole('button', { name: /good response/i }));
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  });

  it('sends the note on Send', async () => {
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" />);
    await user.click(screen.getByRole('button', { name: /good response/i }));
    await user.type(screen.getByRole('textbox'), 'clear and correct');
    await user.click(screen.getByRole('button', { name: /^send$/i }));
    expect(sendFeedback).toHaveBeenLastCalledWith(7, 1, 'clear and correct');
  });

  it('does not re-send when Send is pressed with an empty note', async () => {
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" />);
    await user.click(screen.getByRole('button', { name: /good response/i }));
    expect(sendFeedback).toHaveBeenCalledTimes(1); // the vote itself
    await user.click(screen.getByRole('button', { name: /^send$/i }));
    expect(sendFeedback).toHaveBeenCalledTimes(1); // empty Send is a no-op
  });

  it('sends the note when Enter is pressed in the editor', async () => {
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" />);
    await user.click(screen.getByRole('button', { name: /good response/i }));
    await user.type(screen.getByRole('textbox'), 'looks right{Enter}');
    expect(sendFeedback).toHaveBeenLastCalledWith(7, 1, 'looks right');
  });

  it('calls onRate with the rating after a confirmed vote', async () => {
    const onRate = vi.fn();
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" onRate={onRate} />);
    await user.click(screen.getByRole('button', { name: /bad response/i }));
    await waitFor(() => expect(onRate).toHaveBeenCalledWith(-1));
  });

  it('reflects a prior rating on mount without a click', () => {
    render(<MessageActions messageId={7} content="x" initialRating={-1} />);
    expect(
      screen.getByRole('button', { name: /bad response/i }),
    ).toHaveAttribute('aria-pressed', 'true');
    expect(sendFeedback).not.toHaveBeenCalled();
  });

  it('reverts the thumb and toasts on failure', async () => {
    (sendFeedback as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error('nope'),
    );
    const user = userEvent.setup();
    render(<MessageActions messageId={7} content="x" />);
    const up = screen.getByRole('button', { name: /good response/i });
    await user.click(up);
    await waitFor(() => expect(toast).toHaveBeenCalled());
    expect(up).toHaveAttribute('aria-pressed', 'false');
  });
});
