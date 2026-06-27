import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { Skeleton } from './skeleton';

describe('Skeleton', () => {
  it('renders a pulsing placeholder and merges className', () => {
    const { container } = render(<Skeleton className="h-4 w-20" />);
    const el = container.firstChild as HTMLElement;
    expect(el).toHaveClass('animate-pulse');
    expect(el).toHaveClass('h-4', 'w-20');
    expect(el).toHaveAttribute('aria-hidden', 'true');
  });
});
