// Collects the fields of a periodic operation, then states its effect in plain language before the write happens.

import { useState, type ReactNode } from "react";

import { ActionButton } from "./action-button";
import { ConfirmDialog } from "./confirm-dialog";

type FlowPanelProps = {
  title: string;
  openLabel: string;
  confirmLabel: string;
  summary: string | null;
  busy?: boolean;
  children: ReactNode;
  onConfirm: () => Promise<void>;
};

export function FlowPanel({ title, openLabel, confirmLabel, summary, busy, children, onConfirm }: FlowPanelProps) {
  const [open, setOpen] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  async function confirm() {
    await onConfirm();
    setReviewing(false);
    setOpen(false);
  }
  if (!open) return <ActionButton onClick={() => setOpen(true)}>{openLabel}</ActionButton>;
  return <div className="flow-panel">
    <h4>{title}</h4>
    {children}
    <div className="row-actions">
      <ActionButton variant="text" onClick={() => setOpen(false)}>Anuluj</ActionButton>
      <ActionButton variant="primary" disabled={summary === null} onClick={() => setReviewing(true)}>Dalej</ActionButton>
    </div>
    <ConfirmDialog open={reviewing} title={title} consequence={summary ?? ""} confirmLabel={confirmLabel} busy={busy} onConfirm={() => void confirm()} onCancel={() => setReviewing(false)} />
  </div>;
}
