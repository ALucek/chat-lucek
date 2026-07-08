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

  it('adds the border under border-box so content does not overflow', () => {
    const el = document.createElement('textarea');
    el.style.boxSizing = 'border-box';
    el.style.borderTopWidth = '1px';
    el.style.borderBottomWidth = '1px';
    Object.defineProperty(el, 'scrollHeight', {
      value: 96,
      configurable: true,
    });
    autoSize(el);
    expect(el.style.height).toBe('98px');
  });

  it('returns false on one line and true once content wraps', () => {
    const make = (scrollHeight: number) => {
      const el = document.createElement('textarea');
      el.style.lineHeight = '20px';
      el.style.paddingTop = '0px';
      el.style.paddingBottom = '0px';
      Object.defineProperty(el, 'scrollHeight', {
        value: scrollHeight,
        configurable: true,
      });
      return el;
    };
    expect(autoSize(make(20))).toBe(false);
    expect(autoSize(make(200))).toBe(true);
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
