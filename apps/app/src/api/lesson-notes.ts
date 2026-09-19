// Defines the lesson note contract and requests the teacher note writes and the persona note lists through the shared transport.

import type { PersonaRole } from "./contracts";
import { apiRequest } from "./transport";

export type LessonNote = {
  id: string;
  lesson: string;
  assignment: string;
  lesson_start_at: string;
  body: string;
  materials: { id: string; title: string }[];
  updated_at: string;
};

const part = encodeURIComponent;

export const fetchLessonNotes = (role: PersonaRole, assignmentId: string, signal?: AbortSignal) =>
  apiRequest<{ items: LessonNote[] }>(`/api/${role === "teacher" ? "teachers" : "learners"}/assignments/${part(assignmentId)}/lesson-notes`, { signal });

export const saveLessonNote = (lessonId: string, body: { body: string; materials: string[] }) =>
  apiRequest<LessonNote>(`/api/teachers/lessons/${part(lessonId)}/note`, { method: "PUT", body });

export const deleteLessonNote = (lessonId: string) =>
  apiRequest<void>(`/api/teachers/lessons/${part(lessonId)}/note`, { method: "DELETE" });
