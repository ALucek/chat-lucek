import { cn } from '@/lib/cn';

export function Textarea({
  className,
  ...props
}: React.ComponentPropsWithRef<'textarea'>) {
  return (
    <textarea
      className={cn(
        'border-border bg-surface text-fg placeholder:text-subtle focus-visible:border-accent w-full resize-none rounded-[var(--radius)] border px-3 py-2 text-sm focus-visible:outline-none disabled:opacity-50',
        className,
      )}
      {...props}
    />
  );
}
