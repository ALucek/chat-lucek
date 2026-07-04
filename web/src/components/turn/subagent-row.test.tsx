import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SubagentRow } from './subagent-row';

const node = {
  id: 'SA',
  parent_id: null,
  type: 'tool' as const,
  name: 'run_subagent',
  input: { task: 'research' },
  children: [
    {
      id: 'c1',
      parent_id: 'SA',
      type: 'text' as const,
      text: 'child',
      children: [],
    },
  ],
};

describe('SubagentRow', () => {
  it('is collapsed by default and renders children on expand', async () => {
    render(
      <SubagentRow
        node={node}
        renderChild={(c) => <div key={c.id}>child:{c.id}</div>}
      />,
    );
    expect(screen.getByText('subagent')).toBeInTheDocument();
    expect(screen.getByText('research')).toBeInTheDocument();
    expect(screen.queryByText('child:c1')).toBeNull();
    await userEvent.click(screen.getByRole('button'));
    expect(screen.getByText('child:c1')).toBeInTheDocument();
  });

  it('shows an in-progress dot when active', () => {
    const noop = () => null;
    const { container, rerender } = render(
      <SubagentRow node={node} renderChild={noop} />,
    );
    expect(container.querySelector('.animate-pulse')).toBeNull();
    rerender(<SubagentRow node={node} active renderChild={noop} />);
    expect(container.querySelector('.animate-pulse')).not.toBeNull();
  });
});
