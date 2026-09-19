// Confirms an action the panel cannot undo, naming the consequence with concrete dates or amounts.

import { useEffect, useRef, type ReactNode } from "react";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  consequence: string;
  confirmLabel: string;
  danger?: boolean;
  busy?: boolean;
  children?: ReactNode;
  confirmDisabled?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({ open, title, consequence, confirmLabel, danger, busy, children, confirmDisabled, onConfirm, onCancel }: ConfirmDialogProps) {
  const dialog = useRef<HTMLDivElement>(null);
  const opener = useRef<Element | null>(null);
  useEffect(() => {
    if (!open) return;
    opener.current = document.activeElement;
    dialog.current?.querySelector<HTMLElement>("button, input, select, textarea")?.focus();
    return () => { (opener.current as HTMLElement | null)?.focus?.(); };
  }, [open]);
  if (!open) return null;
  return <div className="dialog-backdrop" onKeyDown={(event) => { if (event.key === "Escape") onCancel(); if (event.key === "Tab") trapFocus(event, dialog.current); }}>
    <div className="dialog" role="dialog" aria-modal="true" aria-label={title} ref={dialog}>
      <h2>{title}</h2>
      <p className="supporting-copy">{consequence}</p>
      {children}
      <div className="row-actions">
        <button className="secondary-button" onClick={onCancel}>Anuluj</button>
        <button className={`primary-button dialog-confirm ${danger ? "danger-button" : ""}`} disabled={busy || confirmDisabled} aria-busy={busy} onClick={onConfirm}>{confirmLabel}</button>
      </div>
    </div>
  </div>;
}

function trapFocus(event: React.KeyboardEvent, container: HTMLElement | null) {
  const focusable = container?.querySelectorAll<HTMLElement>("button:not([disabled]), input, select, textarea");
  if (!focusable || focusable.length === 0) return;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
  if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
}
