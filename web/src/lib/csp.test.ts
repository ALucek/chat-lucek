import { describe, it, expect } from 'vitest';
import { buildCSP } from './csp';

describe('buildCSP', () => {
  it('includes the core directives and the API origin in connect-src', () => {
    const csp = buildCSP('https://api.example.com', false, 'NONCE123');
    expect(csp).toContain("default-src 'self'");
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("object-src 'none'");
    expect(csp).toContain("base-uri 'self'");
    expect(csp).toContain("connect-src 'self' https://api.example.com");
  });

  it('uses a nonce + strict-dynamic in production (no unsafe-eval, no ws)', () => {
    const csp = buildCSP('https://api.example.com', false, 'NONCE123');
    expect(csp).toContain("script-src 'nonce-NONCE123' 'strict-dynamic'");
    expect(csp).not.toContain('unsafe-eval');
    expect(csp).not.toContain('ws:');
  });

  it('relaxes for dev HMR (unsafe-inline + unsafe-eval + ws, no nonce)', () => {
    const csp = buildCSP('http://localhost:8080', true, 'NONCE123');
    expect(csp).toContain('unsafe-eval');
    expect(csp).toContain('ws:');
    expect(csp).not.toContain('nonce-');
  });

  it('allows the Google Identity Services origins', () => {
    const csp = buildCSP('http://localhost:8080', false, 'NONCE123');
    expect(csp).toContain('https://accounts.google.com/gsi/');
  });
});
