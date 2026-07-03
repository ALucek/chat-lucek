import { describe, it, expect, afterEach } from 'vitest';
import { NextRequest } from 'next/server';
import { proxy } from './proxy';

describe('proxy security headers', () => {
  const res = proxy(new NextRequest('http://localhost/'));

  it('sets the static security baseline', () => {
    expect(res.headers.get('X-Content-Type-Options')).toBe('nosniff');
    expect(res.headers.get('X-Frame-Options')).toBe('DENY');
    expect(res.headers.get('Referrer-Policy')).toBe('no-referrer');
    expect(res.headers.get('Permissions-Policy')).toBe(
      'camera=(), microphone=(), geolocation=()',
    );
  });

  it('keeps CSP and HSTS', () => {
    expect(res.headers.get('Content-Security-Policy')).toBeTruthy();
    expect(res.headers.get('Strict-Transport-Security')).toBeTruthy();
  });
});

describe('proxy maintenance mode', () => {
  afterEach(() => {
    delete process.env.MAINTENANCE_MODE;
  });

  it('rewrites to /maintenance when the flag is set', () => {
    process.env.MAINTENANCE_MODE = '1';
    const res = proxy(new NextRequest('http://localhost/c/123'));
    expect(res.headers.get('x-middleware-rewrite')).toContain('/maintenance');
  });

  it('does not rewrite the maintenance page itself', () => {
    process.env.MAINTENANCE_MODE = '1';
    const res = proxy(new NextRequest('http://localhost/maintenance'));
    expect(res.headers.get('x-middleware-rewrite')).toBeNull();
  });

  it('keeps legal pages reachable during maintenance', () => {
    process.env.MAINTENANCE_MODE = '1';
    for (const p of ['/terms', '/privacy']) {
      const res = proxy(new NextRequest(`http://localhost${p}`));
      expect(res.headers.get('x-middleware-rewrite')).toBeNull();
    }
  });

  it('does not rewrite when the flag is off', () => {
    const res = proxy(new NextRequest('http://localhost/'));
    expect(res.headers.get('x-middleware-rewrite')).toBeNull();
  });
});
