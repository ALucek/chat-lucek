import type { NextConfig } from 'next';
import { buildCSP } from './src/lib/csp';

const apiOrigin = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
const csp = buildCSP(apiOrigin, process.env.NODE_ENV !== 'production');

const nextConfig: NextConfig = {
  output: 'standalone',
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [{ key: 'Content-Security-Policy', value: csp }],
      },
    ];
  },
};

export default nextConfig;
