import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CompactionRow } from './compaction-row';

const node = {
  id: 'r1:compaction',
  parent_id: null,
  type: 'compaction' as const,
  text: 'folded summary text',
  children: [],
};

describe('CompactionRow', () => {
  it('shows the fixed label and detail, collapsed by default', () => {
    render(<CompactionRow node={node} />);
    expect(screen.getByText('compacting')).toBeInTheDocument();
    expect(
      screen.getByText('Summarizing earlier conversation…'),
    ).toBeInTheDocument();
    expect(screen.queryByText('folded summary text')).toBeNull();
  });

  it('expands to show the streamed summary', async () => {
    render(<CompactionRow node={node} />);
    await userEvent.click(screen.getByRole('button'));
    expect(screen.getByText('folded summary text')).toBeInTheDocument();
  });

  it('pulses the chip only when active', () => {
    const { container, rerender } = render(<CompactionRow node={node} />);
    expect(container.querySelector('.animate-pulse')).toBeNull();
    rerender(<CompactionRow node={node} active />);
    expect(container.querySelector('.animate-pulse')).not.toBeNull();
  });
});
