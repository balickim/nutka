// Declares learner material contracts and endpoint calls, turning a teacher draft into the multipart upload the backend expects.

import type { PersonaRole } from "./contracts";
import { apiRequest } from "./transport";

export type MaterialAttachment = { name: string; kind: "image" | "pdf"; url: string };
export type Material = { id: string; assignment: string; title: string; body: string; attachments: MaterialAttachment[]; created_at: string };
export type MaterialDraft = { title: string; body: string; files: File[] };

export const materialLimits = { maxFiles: 10, maxFileBytes: 10 * 1024 * 1024, accept: "image/jpeg,image/png,image/webp,image/gif,application/pdf" };

const segment = (role: PersonaRole) => (role === "teacher" ? "teachers" : "learners");

export function fetchMaterials(role: PersonaRole, assignmentId: string, signal?: AbortSignal) {
  return apiRequest<{ items: Material[] }>(`/api/${segment(role)}/assignments/${encodeURIComponent(assignmentId)}/materials`, { signal });
}

export function createMaterial(assignmentId: string, draft: MaterialDraft) {
  const form = new FormData();
  form.append("title", draft.title);
  form.append("body", draft.body);
  draft.files.forEach((file) => form.append("attachments", file));
  return apiRequest<Material>(`/api/teachers/assignments/${encodeURIComponent(assignmentId)}/materials`, { method: "POST", body: form });
}

export function deleteMaterial(id: string) {
  return apiRequest<void>(`/api/teachers/materials/${encodeURIComponent(id)}`, { method: "DELETE" });
}
