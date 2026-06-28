import { describe, it, expect, vi } from 'vitest';

vi.mock('./globals.css', () => ({}));
vi.mock('next/font/google', () => ({
  Share_Tech_Mono: () => ({ variable: 'font-share-tech' }),
}));

import { viewport } from './layout';

describe('root viewport', () => {
  it('hard-locks zoom and fits the device width', () => {
    expect(viewport.width).toBe('device-width');
    expect(viewport.initialScale).toBe(1);
    expect(viewport.maximumScale).toBe(1);
    expect(viewport.userScalable).toBe(false);
    expect(viewport.interactiveWidget).toBe('resizes-content');
  });
});
