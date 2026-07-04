import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToolRow } from './tool-row';

const node = {
  id: 't',
  parent_id: null,
  type: 'tool' as const,
  name: 'internet_search',
  input: { query: 'cats' },
  output: { results: 3 },
  children: [],
};

describe('ToolRow', () => {
  it('shows humanized label + input digest', () => {
    render(<ToolRow node={node} />);
    expect(screen.getByText('search')).toBeInTheDocument();
    expect(screen.getByText('cats')).toBeInTheDocument();
  });

  it('reveals the result in a drawer when expanded', async () => {
    render(<ToolRow node={node} />);
    expect(screen.queryByText(/"results": 3/)).toBeNull();
    await userEvent.click(screen.getByRole('button'));
    expect(screen.getByText(/"results": 3/)).toBeInTheDocument();
  });

  it('flashes the chip only when active', () => {
    const { container, rerender } = render(<ToolRow node={node} />);
    expect(container.querySelector('.animate-pulse')).toBeNull();
    rerender(<ToolRow node={node} active />);
    expect(container.querySelector('.animate-pulse')).not.toBeNull();
  });
});
