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

  it('reveals raw input/output only when expanded', async () => {
    render(<ToolRow node={node} />);
    expect(screen.queryByText('input')).toBeNull();
    await userEvent.click(screen.getByRole('button'));
    expect(screen.getByText('input')).toBeInTheDocument();
    expect(screen.getByText('output')).toBeInTheDocument();
    expect(screen.getByText(/"query": "cats"/)).toBeInTheDocument();
  });
});
