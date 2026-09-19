// Shows learner materials newest first: sanitized rich text, inline images, and PDF links. Callers add a caption line and controls per material.

import type { ReactNode } from "react";

import type { Material, MaterialAttachment } from "../api/materials";
import { apiUrl } from "../api/url";
import { formatScheduleDate } from "../time/schedule";
import { EmptyState } from "./empty-state";

type ListProps = { items: Material[]; empty: string; caption?: (material: Material) => string | null; actions?: (material: Material) => ReactNode };

export function MaterialList({ items, empty, caption, actions }: ListProps) {
  if (items.length === 0) return <EmptyState>{empty}</EmptyState>;
  return <div className="material-list">{items.map((material) => <article className="material-card" key={material.id}>
    <div className="material-heading">
      <div>{caption?.(material) ? <p className="eyebrow">{caption(material)}</p> : null}<h3>{material.title}</h3><p className="lesson-meta">Dodano {formatScheduleDate(material.created_at)}</p></div>
      {actions ? <div className="row-actions">{actions(material)}</div> : null}
    </div>
    {/* The backend stores only sanitized HTML, so rendering it cannot run scripts. */}
    {material.body ? <div className="material-body" dangerouslySetInnerHTML={{ __html: material.body }} /> : null}
    <Attachments items={material.attachments} />
  </article>)}</div>;
}

function Attachments({ items }: { items: MaterialAttachment[] }) {
  const images = items.filter((item) => item.kind === "image");
  const documents = items.filter((item) => item.kind === "pdf");
  return <>
    {images.length ? <div className="material-images">{images.map((item) => <a key={item.name} href={apiUrl(item.url)} target="_blank" rel="noopener"><img src={apiUrl(item.url)} alt={displayName(item.name)} loading="lazy" /></a>)}</div> : null}
    {documents.length ? <ul className="material-files">{documents.map((item) => <li key={item.name}><a href={apiUrl(item.url)} target="_blank" rel="noopener">PDF · {displayName(item.name)}</a></li>)}</ul> : null}
  </>;
}

// PocketBase appends a random ten-character suffix to every stored file name.
function displayName(name: string): string {
  return name.replace(/_[a-z0-9]{10}(\.[^.]+)$/i, "$1");
}
