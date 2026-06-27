'use client';

import './globals.css';
import { ErrorFallback } from '@/components/error-fallback';

export default function GlobalError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en" className="h-full">
      <body className="flex min-h-full flex-col">
        <ErrorFallback onReset={reset} />
      </body>
    </html>
  );
}
