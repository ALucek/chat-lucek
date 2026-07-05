import { describe, it, expect } from 'vitest';
import { autoSize } from './autosize';

describe('autoSize', () => {
  it('sets the textarea height to its scroll height', () => {
    const el = document.createElement('textarea');
    Object.defineProperty(el, 'scrollHeight', {
      value: 96,
      configurable: true,
    });
    autoSize(el);
    expect(el.style.height).toBe('96px');
  });

  it('resets height to auto before measuring so it can shrink', () => {
    const el = document.createElement('textarea');
    // scrollHeight reads the current inline height, proving we reset first.
    Object.defineProperty(el, 'scrollHeight', {
      get() {
        return this.style.height === 'auto' ? 40 : 999;
      },
    });
    el.style.height = '999px';
    autoSize(el);
    expect(el.style.height).toBe('40px');
  });
});
