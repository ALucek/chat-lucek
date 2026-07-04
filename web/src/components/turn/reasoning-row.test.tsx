import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReasoningRow } from './reasoning-row';

const node = {
  id: 'r',
  parent_id: null,
  type: 'reasoning' as const,
  text: 'why the sky is blue',
  children: [],
};

describe('ReasoningRow', () => {
  it('is collapsed by default and expands on click', async () => {
    render(<ReasoningRow node={node} />);
    const btn = screen.getByRole('button');
    expect(btn).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByText('thinking')).toBeInTheDocument();
    await userEvent.click(btn);
    expect(btn).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('why the sky is blue')).toBeInTheDocument();
  });
});
