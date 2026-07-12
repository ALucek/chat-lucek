'use client';

import { useEffect, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

interface DialogProps {
  open: boolean;
  onClose: () => void;
  label: string;
  children: ReactNode;
}

const FOCUSABLE =
  'textarea, button, [href], input, [tabindex]:not([tabindex="-1"])';

// Centered modal; unlike Menu's bottom sheet it works with the mobile keyboard.
export function Dialog({ open, onClose, label, children }: DialogProps) {
  const surfaceRef = useRef<HTMLDivElement>(null);
  const openerRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!open) return;
    openerRef.current = document.activeElement as HTMLElement | null;
    surfaceRef.current?.querySelector<HTMLElement>(FOCUSABLE)?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('keydown', onKey);
      openerRef.current?.focus?.();
    };
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label={label}
    >
      <div
        data-testid="dialog-backdrop"
        onClick={onClose}
        className="absolute inset-0 bg-black/40"
      />
      <div
        ref={surfaceRef}
        className="border-border bg-surface relative z-10 w-full max-w-sm rounded-[var(--radius)] border p-4 shadow-lg"
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}
