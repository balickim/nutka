// Hides a corrective operation behind a disclosure and holds the reason that every correction must record.

import { useId, useState, type ReactNode } from "react";

export function AdvancedOperations({ label = "Operacje zaawansowane", children }: { label?: string; children: (reason: string) => ReactNode }) {
  const [reason, setReason] = useState("");
  const id = useId();
  return <details className="advanced-operations">
    <summary>{label}</summary>
    <p className="supporting-copy">Korekta zapisuje się w historii razem z powodem.</p>
    <label htmlFor={id}>Powód korekty</label>
    <input id={id} value={reason} onChange={(event) => setReason(event.target.value)} required />
    {children(reason)}
  </details>;
}
