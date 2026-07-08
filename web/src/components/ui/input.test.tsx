import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Input } from './input';

describe('Input primitive', () => {
  it('merges a className override onto the base styles', () => {
    render(<Input placeholder="X" className="w-10" />);
    expect(screen.getByPlaceholderText('X')).toHaveClass('w-10');
  });
});
