export function buildCSP(apiOrigin: string, dev: boolean): string {
  return [
    `default-src 'self'`,
    `script-src 'self' 'unsafe-inline'${dev ? " 'unsafe-eval'" : ''}`,
    `style-src 'self' 'unsafe-inline'`,
    `img-src 'self' data:`,
    `connect-src 'self' ${apiOrigin}${dev ? ' ws:' : ''}`,
    `font-src 'self'`,
    `object-src 'none'`,
    `base-uri 'self'`,
    `form-action 'self'`,
    `frame-ancestors 'none'`,
  ].join('; ');
}
