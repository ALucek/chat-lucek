import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TextBlock } from './text-block';

describe('TextBlock', () => {
  it('renders top-level answer markdown with a streaming caret', () => {
    const node = {
      id: 'a',
      parent_id: null,
      type: 'text' as const,
      text: '**hi**',
      children: [],
    };
    const { container } = render(<TextBlock node={node} streaming />);
    expect(screen.getByText('hi')).toBeInTheDocument();
    expect(container.querySelector('.animate-blink')).not.toBeNull();
  });

  it('renders nested subagent text without a caret', () => {
    const node = {
      id: 'c',
      parent_id: 'SA',
      type: 'text' as const,
      text: 'sub',
      children: [],
    };
    const { container } = render(<TextBlock node={node} nested streaming />);
    expect(screen.getByText('sub')).toBeInTheDocument();
    expect(container.querySelector('.animate-blink')).toBeNull();
  });
});
