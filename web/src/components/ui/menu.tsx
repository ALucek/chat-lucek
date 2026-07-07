'use client';

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
  type Ref,
} from 'react';
import { createPortal } from 'react-dom';
import { useMediaQuery } from '@/lib/use-media-query';

interface TriggerProps {
  ref: Ref<HTMLButtonElement>;
  onClick: () => void;
  'aria-haspopup': 'menu';
  'aria-expanded': boolean;
}

interface MenuProps {
  label: string;
  trigger: (props: TriggerProps) => ReactNode;
  children: (api: { close: () => void }) => ReactNode;
}

const FOCUSABLE = 'button, [href], input, [tabindex]:not([tabindex="-1"])';

export function Menu({ label, trigger, children }: MenuProps) {
  const [open, setOpen] = useState(false);
  const [render, setRender] = useState(false);
  const [shown, setShown] = useState(false);
  const [coords, setCoords] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const surfaceRef = useRef<HTMLDivElement>(null);
  const wasOpen = useRef(false);
  const isDesktop = useMediaQuery('(min-width: 768px)');

  // Mount, then flip to shown next frame so the enter transition runs.
  const openMenu = useCallback(() => {
    setOpen(true);
    setRender(true);
    requestAnimationFrame(() => setShown(true));
  }, []);

  const close = useCallback(() => {
    setOpen(false);
    setShown(false);
  }, []);

  // Hold the mobile sheet mounted until its exit transition finishes.
  useEffect(() => {
    if (isDesktop || open || !render) return;
    const t = setTimeout(() => setRender(false), 200);
    return () => clearTimeout(t);
  }, [isDesktop, open, render]);

  // Anchor the desktop popover under the trigger's bottom-right corner.
  useLayoutEffect(() => {
    if (!open || !isDesktop || !triggerRef.current) return;
    const r = triggerRef.current.getBoundingClientRect();
    setCoords({ top: r.bottom + 4, left: r.right });
  }, [open, isDesktop]);

  // Focus into the surface on open; restore focus to the trigger on close.
  useEffect(() => {
    if (open) {
      wasOpen.current = true;
      surfaceRef.current?.querySelector<HTMLElement>(FOCUSABLE)?.focus();
    } else if (wasOpen.current) {
      wasOpen.current = false;
      triggerRef.current?.focus();
    }
  }, [open]);

  // Dismiss on Escape or outside click; desktop also closes on scroll/resize.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
      if (e.key === 'Tab') trapFocus(e, surfaceRef.current);
    };
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (surfaceRef.current?.contains(t)) return;
      if (triggerRef.current?.contains(t)) return;
      close();
    };
    document.addEventListener('keydown', onKey);
    document.addEventListener('mousedown', onDown);
    let cleanupDesktop = () => {};
    if (isDesktop) {
      window.addEventListener('scroll', close, true);
      window.addEventListener('resize', close);
      cleanupDesktop = () => {
        window.removeEventListener('scroll', close, true);
        window.removeEventListener('resize', close);
      };
    }
    return () => {
      document.removeEventListener('keydown', onKey);
      document.removeEventListener('mousedown', onDown);
      cleanupDesktop();
    };
  }, [open, isDesktop, close]);

  const surface = isDesktop ? (
    <div
      ref={surfaceRef}
      role="menu"
      aria-label={label}
      style={{ top: coords.top, left: coords.left }}
      className="border-border bg-surface fixed z-50 min-w-[168px] -translate-x-full rounded-[var(--radius)] border p-1 shadow-lg"
    >
      {children({ close })}
    </div>
  ) : (
    <>
      <div
        data-testid="menu-backdrop"
        onClick={close}
        className={`fixed inset-0 z-40 bg-black/40 transition-opacity duration-200 ${
          shown ? 'opacity-100' : 'opacity-0'
        }`}
      />
      <div
        ref={surfaceRef}
        role="dialog"
        aria-label={label}
        className={`border-border bg-surface fixed inset-x-0 bottom-0 z-50 rounded-t-xl border-t p-2 pb-[calc(0.5rem+env(safe-area-inset-bottom))] transition-transform duration-200 ${
          shown ? 'translate-y-0' : 'translate-y-full'
        }`}
      >
        {children({ close })}
      </div>
    </>
  );

  return (
    <>
      {trigger({
        ref: triggerRef,
        onClick: () => (open ? close() : openMenu()),
        'aria-haspopup': 'menu',
        'aria-expanded': open,
      })}
      {(isDesktop ? open : render) && createPortal(surface, document.body)}
    </>
  );
}

// Keep Tab focus inside the surface while it is open.
function trapFocus(e: KeyboardEvent, surface: HTMLElement | null) {
  if (!surface) return;
  const items = surface.querySelectorAll<HTMLElement>(FOCUSABLE);
  if (items.length === 0) return;
  const first = items[0];
  const last = items[items.length - 1];
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault();
    first.focus();
  }
}
