// Discloses the business rule that governs a field, next to that field instead of above the form.

import { useId, useState } from "react";

export function PolicyHint({ children }: { children: string }) {
  const [open, setOpen] = useState(false);
  const id = useId();
  return <span className="policy-hint">
    <button type="button" className="policy-hint-toggle" aria-expanded={open} aria-controls={id} aria-label="Pokaż zasadę" onClick={() => setOpen(!open)}>?</button>
    {open ? <span className="policy-hint-body" id={id}>{children}</span> : null}
  </span>;
}
