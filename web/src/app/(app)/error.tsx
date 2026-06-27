'use client';

import { ErrorFallback } from '@/components/error-fallback';

export default function AppError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return <ErrorFallback onReset={reset} />;
}
