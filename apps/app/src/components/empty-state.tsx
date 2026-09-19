// States that a block holds nothing and, when an action can change that, offers the next step.

import type { ReactNode } from "react";

export function EmptyState({ children, action }: { children: string; action?: ReactNode }) {
  return <div className="empty-state"><p>{children}</p>{action}</div>;
}
