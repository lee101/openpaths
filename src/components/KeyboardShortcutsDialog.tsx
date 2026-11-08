import { Keyboard, X } from 'lucide-react';
import { useDialog } from '../lib/useDialog';

export function KeyboardShortcutsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const dialogRef = useDialog({ open, onClose });
  if (!open) return null;

  const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.platform);
  const mod = isMac ? '⌘' : 'Ctrl';
  const rows: Array<{ keys: string[]; label: string }> = [
    { keys: [mod, 'K'], label: 'Open search' },
    { keys: ['/'], label: 'Focus the page search box' },
    { keys: ['?'], label: 'Show this dialog' },
    { keys: [mod, 'Enter'], label: 'Send or run' },
    { keys: ['Shift', 'Enter'], label: 'New line in a message' },
    { keys: ['Esc'], label: 'Close dialog, menu, or panel' },
    { keys: ['Tab'], label: 'Move through controls' },
    { keys: ['Shift', 'Tab'], label: 'Move backwards' },
  ];

  return (
    <div
      className="fixed inset-0 z-[70] flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm"
      onMouseDown={event => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={dialogRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby="keyboard-shortcuts-title"
        className="w-full max-w-lg rounded-2xl border border-white/20 bg-[#090909] p-6 shadow-[0_24px_80px_rgba(0,0,0,0.55)] outline-none"
        data-testid="keyboard-shortcuts"
      >
        <div className="mb-5 flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-lg border border-white/20 bg-white/[0.07]">
              <Keyboard className="h-5 w-5 text-white/70" />
            </span>
            <div>
              <h2 id="keyboard-shortcuts-title" className="text-lg font-semibold tracking-tight">Keyboard shortcuts</h2>
              <p className="text-xs font-mono text-white/45">Move faster without leaving the keys.</p>
            </div>
          </div>
          <button type="button" onClick={onClose} aria-label="Close keyboard shortcuts" className="rounded-lg p-2 text-white/55 hover:bg-white/10 hover:text-white">
            <X className="h-5 w-5" />
          </button>
        </div>

        <dl className="divide-y divide-white/10 border-y border-white/10">
          {rows.map(row => (
            <div key={`${row.keys.join('+')}-${row.label}`} className="flex items-center justify-between gap-4 py-2.5">
              <dt className="text-sm text-white/70">{row.label}</dt>
              <dd className="flex shrink-0 items-center gap-1">
                {row.keys.map(key => (
                  <kbd key={key} className="rounded border border-white/25 bg-white/10 px-2 py-0.5 font-mono text-[11px] text-white/80">{key}</kbd>
                ))}
              </dd>
            </div>
          ))}
        </dl>

        <p className="mt-4 text-xs leading-5 text-white/45">
          Shortcuts work outside text fields. <span className="font-mono text-white/60">{mod}K</span> and <span className="font-mono text-white/60">{mod}Enter</span> work everywhere.
        </p>
      </div>
    </div>
  );
}
