import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PlanDock } from './plan-dock';
import type { RunNode } from '@/lib/api';

function planNodes(todos: unknown): RunNode[] {
  return [
    {
      id: '1',
      parent_id: null,
      type: 'tool',
      name: 'set_todos',
      input: { todos },
    },
  ];
}

// In-progress item shows in the header; the outstanding item only in the body.
const TODOS = [
  { description: 'Draft schema', progress: 'in progress' },
  { description: 'Write tests', progress: 'outstanding' },
];

describe('PlanDock', () => {
  it('renders nothing when there is no plan', () => {
    const { container } = render(<PlanDock nodes={[]} running={false} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders a row per todo and the done/total count when expanded', () => {
    render(
      <PlanDock
        nodes={planNodes([
          { description: 'Research', progress: 'completed' },
          { description: 'Draft', progress: 'in progress' },
          { description: 'Test', progress: 'outstanding' },
        ])}
        running
      />,
    );
    expect(screen.getAllByRole('listitem')).toHaveLength(3);
    expect(screen.getByText('Research')).toBeInTheDocument();
    expect(screen.getByText('Test')).toBeInTheDocument();
    expect(screen.getByText('1/3')).toBeInTheDocument();
  });

  it('is collapsed when idle and expanded when running', () => {
    const { rerender } = render(
      <PlanDock nodes={planNodes(TODOS)} running={false} />,
    );
    expect(screen.queryByText('Write tests')).not.toBeInTheDocument();
    expect(screen.getByText('0/2')).toBeInTheDocument();
    rerender(<PlanDock nodes={planNodes(TODOS)} running />);
    expect(screen.getByText('Write tests')).toBeInTheDocument();
  });

  it('toggles the body when the header is clicked', async () => {
    const user = userEvent.setup();
    render(<PlanDock nodes={planNodes(TODOS)} running={false} />);
    expect(screen.queryByText('Write tests')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button'));
    expect(screen.getByText('Write tests')).toBeInTheDocument();
  });
});
