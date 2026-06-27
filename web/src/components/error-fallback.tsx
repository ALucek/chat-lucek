import { Button } from '@/components/ui/button';

export function ErrorFallback({ onReset }: { onReset: () => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-6 text-center">
      <h1 className="text-fg text-lg font-semibold">Something went wrong</h1>
      <p className="text-muted text-sm">
        An unexpected error occurred. You can try again.
      </p>
      <Button onClick={onReset}>Try again</Button>
    </div>
  );
}
