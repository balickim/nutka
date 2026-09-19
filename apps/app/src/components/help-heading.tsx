// Pairs a section heading with a "?" tooltip that explains the section in everyday words.

import { Tooltip } from "./tooltip";

export function HelpHeading({ title, help, level = 2 }: { title: string; help: string; level?: 2 | 4 }) {
  const Heading = level === 4 ? "h4" : "h2";
  return <div className="heading-with-help"><Heading>{title}</Heading><Tooltip label={`Co oznacza „${title}”?`} text={help} /></div>;
}
