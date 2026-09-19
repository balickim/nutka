// Runs a write from a button that keeps its label and size while the write is in flight.

import type { ReactNode } from "react";

type Variant = "primary" | "secondary" | "text";

const variantClass: Record<Variant, string> = { primary: "btn btn-primary btn-sm", secondary: "btn btn-ghost btn-sm", text: "text-button" };

export function ActionButton({ busy, variant = "secondary", danger, children, ...rest }: { busy?: boolean; variant?: Variant; danger?: boolean; children: ReactNode } & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const className = `${variantClass[variant]}${danger ? " danger-button" : ""}${busy ? " is-busy" : ""}`;
  return <button {...rest} className={className} aria-busy={busy} disabled={rest.disabled || busy}>{children}{busy ? <span className="spinner" aria-hidden="true" /> : null}</button>;
}
