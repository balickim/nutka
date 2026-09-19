// Edits a short formatted text with a minimal toolbar and reports its HTML. The backend sanitizes that HTML before storage.

import { useRef } from "react";

const commands = [
  { label: "B", title: "Pogrubienie", command: "bold" },
  { label: "I", title: "Kursywa", command: "italic" },
  { label: "U", title: "Podkreślenie", command: "underline" },
  { label: "Nagłówek", title: "Nagłówek", command: "formatBlock", value: "h3" },
  { label: "Akapit", title: "Zwykły akapit", command: "formatBlock", value: "p" },
  { label: "• Lista", title: "Lista punktowana", command: "insertUnorderedList" },
  { label: "1. Lista", title: "Lista numerowana", command: "insertOrderedList" },
] as const;

export function RichTextEditor({ label, onChange }: { label: string; onChange: (html: string) => void }) {
  const area = useRef<HTMLDivElement>(null);
  function run(command: string, value?: string) {
    area.current?.focus();
    // execCommand is deprecated but remains the only dependency-free formatting API for contentEditable.
    document.execCommand(command, false, value);
    onChange(area.current?.innerHTML ?? "");
  }
  return <div className="rich-text">
    <div className="rich-text-toolbar" role="toolbar" aria-label={`Formatowanie: ${label}`}>
      {commands.map((item) => <button key={item.title} type="button" title={item.title} onMouseDown={(event) => event.preventDefault()} onClick={() => run(item.command, "value" in item ? item.value : undefined)}>{item.label}</button>)}
    </div>
    <div ref={area} className="rich-text-area material-body" contentEditable suppressContentEditableWarning role="textbox" aria-multiline="true" aria-label={label} onInput={(event) => onChange(event.currentTarget.innerHTML)} />
  </div>;
}
