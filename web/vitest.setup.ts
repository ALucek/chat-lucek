import '@testing-library/jest-dom/vitest';

// jsdom doesn't implement ResizeObserver, which use-stick-to-bottom needs.
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// jsdom localStorage; don't drop --no-experimental-webstorage in test script.
