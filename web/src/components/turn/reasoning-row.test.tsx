import { describe, it, expect, afterEach } from 'vitest';
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

// jsdom has no layout, so drive the overflow measurement directly.
function mockWidths(scroll: number, client: number) {
  Object.defineProperty(HTMLElement.prototype, 'scrollWidth', {
    configurable: true,
    get: () => scroll,
  });
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
    configurable: true,
    get: () => client,
  });
}

describe('ReasoningRow', () => {
  afterEach(() => {
    delete (HTMLElement.prototype as { scrollWidth?: number }).scrollWidth;
    delete (HTMLElement.prototype as { clientWidth?: number }).clientWidth;
  });

  it('expands to the full thought when the line overflows', async () => {
    mockWidths(200, 50);
    render(<ReasoningRow node={node} />);
    const btn = screen.getByRole('button');
    expect(btn).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByText('thinking')).toBeInTheDocument();
    expect(screen.getAllByText('why the sky is blue')).toHaveLength(1);
    await userEvent.click(btn);
    expect(btn).toHaveAttribute('aria-expanded', 'true');
    // the drawer now shows the thought a second time, in full
    expect(screen.getAllByText('why the sky is blue')).toHaveLength(2);
  });

  it('shows no expander when the thought fits on one line', () => {
    mockWidths(50, 200);
    render(<ReasoningRow node={node} />);
    expect(screen.queryByRole('button')).toBeNull();
    expect(screen.getByText('thinking')).toBeInTheDocument();
    expect(screen.queryByText('▸')).toBeNull();
  });
});
