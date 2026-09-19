// Explains a nearby label through a "?" icon that reveals the text on hover or focus. Touch users tap the icon.

import { useId } from "react";

export function Tooltip({ text, label = "Co to jest?", align = "start" }: { text: string; label?: string; align?: "start" | "end" }) {
  const id = useId();
  return <span className="tooltip">
    <button type="button" className="tooltip-trigger" aria-label={label} aria-describedby={id}>?</button>
    <span className={`tooltip-body tooltip-${align}`} role="tooltip" id={id}>{text}</span>
  </span>;
}
