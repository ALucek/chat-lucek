export function buildCSP(
  apiOrigin: string,
  dev: boolean,
  nonce: string,
): string {
  const gsi = 'https://accounts.google.com/gsi/';
  const wellKnown = 'https://accounts.google.com/.well-known/';
  const oauth = 'https://accounts.google.com/o/oauth2/';
  const script = dev
    ? `script-src 'self' 'unsafe-inline' 'unsafe-eval' ${gsi}client`
    : `script-src 'nonce-${nonce}' 'strict-dynamic'`;
  return [
    `default-src 'self'`,
    script,
    `style-src 'self' 'unsafe-inline' ${gsi}style`,
    `img-src 'self' data:`,
    `connect-src 'self' ${apiOrigin} ${gsi} ${wellKnown}${dev ? ' ws:' : ''}`,
    `font-src 'self'`,
    `frame-src ${gsi} ${oauth}`,
    `object-src 'none'`,
    `base-uri 'self'`,
    `form-action 'self'`,
    `frame-ancestors 'none'`,
  ].join('; ');
}
