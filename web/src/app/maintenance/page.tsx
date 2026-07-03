import Link from 'next/link';
import { Wordmark } from '@/components/wordmark';

export default function MaintenancePage() {
  return (
    <main className="bg-bg flex min-h-dvh items-center justify-center p-6">
      <div className="flex flex-col items-center gap-6">
        <Wordmark />
        <div className="border-border bg-surface flex w-full max-w-sm flex-col items-center gap-4 rounded-[var(--radius)] border p-8">
          <p className="text-fg text-center text-sm">Down for maintenance.</p>
          <div className="bg-border h-px w-full" />
          <p className="text-subtle text-center text-xs leading-relaxed">
            By continuing you agree to the{' '}
            <Link
              href="/terms"
              target="_blank"
              rel="noopener noreferrer"
              className="underline"
            >
              Terms
            </Link>{' '}
            &amp;{' '}
            <Link
              href="/privacy"
              target="_blank"
              rel="noopener noreferrer"
              className="underline"
            >
              Privacy Policy
            </Link>
            .
          </p>
        </div>
        <p className="text-subtle text-xs">
          Made by{' '}
          <a
            href="https://lucek.ai"
            target="_blank"
            rel="noopener noreferrer"
            className="underline"
          >
            Adam Lucek
          </a>
        </p>
      </div>
    </main>
  );
}
