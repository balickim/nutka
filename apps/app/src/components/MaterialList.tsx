// Shows learner materials newest first: sanitized rich text, inline images, and PDF links. The teacher view adds a delete control.

import type { Material, MaterialAttachment } from "../api/materials";
import { apiUrl } from "../api/url";
import { formatScheduleDate } from "../time/schedule";
import { EmptyState } from "./EmptyState";

export function MaterialList({ items, empty, onDelete }: { items: Material[]; empty: string; onDelete?: (material: Material) => void }) {
  if (items.length === 0) return <EmptyState>{empty}</EmptyState>;
  return <div className="material-list">{items.map((material) => <article className="material-card" key={material.id}>
    <div className="material-heading">
      <div><h3>{material.title}</h3><p className="lesson-meta">Dodano {formatScheduleDate(material.created_at)}</p></div>
      {onDelete ? <button className="text-button danger-button" onClick={() => onDelete(material)}>Usuń</button> : null}
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
