import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Wordmark } from './wordmark';

describe('Wordmark', () => {
  it('renders labelled ascii art in a <pre>', () => {
    render(<Wordmark />);
    expect(screen.getByLabelText('Chat Łucek').tagName).toBe('PRE');
  });
});
