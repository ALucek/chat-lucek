import '@testing-library/jest-dom/vitest';

// jsdom doesn't implement ResizeObserver, which use-stick-to-bottom needs.
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// jsdom localStorage; don't drop --no-experimental-webstorage in test script.

// jsdom lacks matchMedia; stub what useMediaQuery reads (mobile-first).
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }) as unknown as MediaQueryList;
}
