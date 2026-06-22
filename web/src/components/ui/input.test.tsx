import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Input } from './input';
import { Textarea } from './textarea';

describe('Input/Textarea primitives', () => {
  it('Input forwards placeholder and type', () => {
    render(<Input type="email" placeholder="Email" />);
    const el = screen.getByPlaceholderText('Email');
    expect(el).toHaveAttribute('type', 'email');
  });

  it('Textarea renders as a textbox and forwards disabled', () => {
    render(<Textarea disabled placeholder="Send a message…" />);
    expect(screen.getByRole('textbox')).toBeDisabled();
  });

  it('merges a className override', () => {
    render(<Input placeholder="X" className="w-10" />);
    expect(screen.getByPlaceholderText('X')).toHaveClass('w-10');
  });
});
