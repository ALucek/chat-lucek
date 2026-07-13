import { NextRequest, NextResponse } from 'next/server';
import { buildCSP } from '@/lib/csp';

// Empty = same-origin, so CSP connect-src stays 'self'; set for cross-origin.
const API_URL = process.env.NEXT_PUBLIC_API_URL ?? '';

// Reachable even during maintenance: the page itself and its legal links.
const MAINTENANCE_EXEMPT = new Set(['/maintenance', '/terms', '/privacy']);

export function proxy(request: NextRequest) {
  if (
    process.env.MAINTENANCE_MODE === '1' &&
    !MAINTENANCE_EXEMPT.has(request.nextUrl.pathname)
  ) {
    return NextResponse.rewrite(new URL('/maintenance', request.url));
  }

  const dev = process.env.NODE_ENV !== 'production';
  const nonce = btoa(crypto.randomUUID());
  const csp = buildCSP(API_URL, dev, nonce);

  // Pass nonce + policy on the request so Next stamps its inline scripts.
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set('x-nonce', nonce);
  requestHeaders.set('Content-Security-Policy', csp);

  const response = NextResponse.next({ request: { headers: requestHeaders } });
  response.headers.set('Content-Security-Policy', csp);
  response.headers.set(
    'Strict-Transport-Security',
    'max-age=31536000; includeSubDomains',
  );
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('Referrer-Policy', 'no-referrer');
  response.headers.set(
    'Permissions-Policy',
    'camera=(), microphone=(), geolocation=()',
  );
  return response;
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
