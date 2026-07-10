'use client';

import { Suspense, useEffect, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/lib/auth-context';
import { Wordmark } from '@/components/wordmark';

function Verify() {
  const { verifyMagicLink } = useAuth();
  const router = useRouter();
  const token = useSearchParams().get('token');
  const [error, setError] = useState(!token);
  const started = useRef(false);

  useEffect(() => {
    if (started.current || !token) return;
    started.current = true;
    verifyMagicLink(token)
      .then(() => router.replace('/'))
      .catch(() => setError(true));
  }, [token, router, verifyMagicLink]);

  if (error) {
    return (
      <div className="flex flex-col items-center gap-3">
        <p role="alert" className="text-danger text-sm">
          This sign-in link is invalid or expired.
        </p>
        <Link href="/login" className="text-sm underline">
          Back to sign in
        </Link>
      </div>
    );
  }
  return <p className="text-subtle text-sm">Signing you in…</p>;
}

export default function MagicPage() {
  return (
    <main className="bg-bg flex min-h-dvh items-center justify-center p-6">
      <div className="flex flex-col items-center gap-6">
        <Wordmark />
        <Suspense
          fallback={<p className="text-subtle text-sm">Signing you in…</p>}
        >
          <Verify />
        </Suspense>
      </div>
    </main>
  );
}
