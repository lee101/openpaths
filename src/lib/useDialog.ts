import { useEffect, useRef, type RefObject } from 'react';

type DialogEntry = {
  id: symbol;
  closeOnEscape: boolean;
  onClose: () => void;
};

type Options = {
  open: boolean;
  onClose: () => void;
  /** Element to focus when the dialog opens. Falls back to [data-autofocus], then the first focusable child. */
  initialFocusRef?: RefObject<HTMLElement | null>;
  closeOnEscape?: boolean;
};

const FOCUSABLE = [
  'a[href]',
  'area[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'iframe',
  'object',
  'embed',
  'summary',
  '[contenteditable="true"]',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

// Only the topmost dialog reacts to Escape/Tab. Nested dialogs (auth over a
// paywall, Stripe over billing) therefore unwind one layer at a time.
const dialogStack: DialogEntry[] = [];
let openDialogCount = 0;
let bodyOverflowBeforeDialogs = '';

function focusableElements(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(element => {
    if (element.hidden || element.getAttribute('aria-hidden') === 'true') return false;
    if (element.closest('[inert]')) return false;
    return element.getClientRects().length > 0;
  });
}

/**
 * Keyboard and focus lifecycle for modal dialogs: Escape closes the topmost
 * dialog, Tab is trapped, focus is restored on close, and page scroll is locked.
 */
export function useDialog({ open, onClose, initialFocusRef, closeOnEscape = true }: Options): RefObject<HTMLDivElement | null> {
  const containerRef = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  const initialFocusRefRef = useRef(initialFocusRef);
  initialFocusRefRef.current = initialFocusRef;

  useEffect(() => {
    if (!open) return;

    const id = Symbol('dialog');
    const previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const entry: DialogEntry = { id, closeOnEscape, onClose: () => onCloseRef.current() };
    dialogStack.push(entry);

    openDialogCount += 1;
    if (openDialogCount === 1) {
      bodyOverflowBeforeDialogs = document.body.style.overflow;
      document.body.style.overflow = 'hidden';
    }

    const focusInitial = () => {
      const root = containerRef.current;
      if (!root) return;
      const explicit = initialFocusRefRef.current?.current;
      const marked = root.querySelector<HTMLElement>('[data-autofocus]');
      (explicit ?? marked ?? focusableElements(root)[0] ?? root).focus({ preventScroll: true });
    };
    const raf = requestAnimationFrame(focusInitial);

    const onKeyDown = (event: KeyboardEvent) => {
      if (dialogStack[dialogStack.length - 1]?.id !== id) return;

      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopImmediatePropagation();
        if (closeOnEscape) entry.onClose();
        return;
      }

      if (event.key !== 'Tab') return;
      const root = containerRef.current;
      if (!root) return;

      const focusables = focusableElements(root);
      if (focusables.length === 0) {
        event.preventDefault();
        root.focus({ preventScroll: true });
        return;
      }

      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement as HTMLElement | null;

      if (event.shiftKey) {
        if (!active || active === first || !root.contains(active)) {
          event.preventDefault();
          last.focus({ preventScroll: true });
        }
      } else if (!active || active === last || !root.contains(active)) {
        event.preventDefault();
        first.focus({ preventScroll: true });
      }
    };

    document.addEventListener('keydown', onKeyDown, true);
    return () => {
      cancelAnimationFrame(raf);
      document.removeEventListener('keydown', onKeyDown, true);

      const index = dialogStack.findIndex(item => item.id === id);
      if (index >= 0) dialogStack.splice(index, 1);
      openDialogCount = Math.max(0, openDialogCount - 1);
      if (openDialogCount === 0) document.body.style.overflow = bodyOverflowBeforeDialogs;

      if (previouslyFocused?.isConnected) previouslyFocused.focus({ preventScroll: true });
    };
  }, [open, closeOnEscape]);

  return containerRef;
}
