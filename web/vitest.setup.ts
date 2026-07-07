import '@testing-library/jest-dom/vitest';

// jsdom doesn't implement ResizeObserver, which use-stick-to-bottom needs.
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// jsdom localStorage; don't drop --no-experimental-webstorage in test script.

// jsdom lacks matchMedia; default to no-match (mobile-first). Tests override.
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList;
}
