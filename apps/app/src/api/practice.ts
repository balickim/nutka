// Declares the practice task, session, and summary contracts and requests the practice routes of both personas.

import type { PersonaRole } from "./contracts";
import { apiRequest } from "./transport";

export type PracticeLink = { id: string; title: string };
export type TaskStatus = "active" | "done" | "archived";

export type PracticeTask = {
  id: string;
  assignment: string;
  lesson: string | null;
  title: string;
  details: string;
  suggested_minutes: number | null;
  piece: PracticeLink | null;
  material: PracticeLink | null;
  status: TaskStatus;
  position: number;
  created_at: string;
};

export type PracticeSession = { id: string; practiced_on: string; minutes: number | null; tasks: PracticeLink[]; comment: string; created_at: string; deletable: boolean };

export type PracticeSummary = {
  since_on: string;
  days: number;
  minutes: number;
  sessions: number;
  tasks: { task: string; title: string; count: number }[];
  comments: { practiced_on: string; comment: string }[];
};

export type AssignmentPracticeSummary = PracticeSummary & { today: string; recent_days: string[] };
export type LessonPracticeSummary = { lesson: string; assignment: string; summary: PracticeSummary };

export type NewTask = { title: string; details: string; suggested_minutes: number | null; piece: string; material: string };
export type PracticePlan = { lesson?: string; keep: string[]; done: string[]; archive: string[]; create: NewTask[] };
export type SessionDraft = { practiced_on: string; minutes: number | null; tasks: string[]; comment: string };

export const practiceLimits = { titleMax: 200, detailsMax: 1000, commentMax: 500, suggestedMinutesMax: 120, sessionMinutesMax: 240, backfillDays: 14 };

const part = encodeURIComponent;
const base = (role: PersonaRole, assignmentId: string) => `/api/${role === "teacher" ? "teachers" : "learners"}/assignments/${part(assignmentId)}`;

export const fetchPracticeTasks = (role: PersonaRole, assignmentId: string, status: TaskStatus | "all", signal?: AbortSignal) =>
  apiRequest<{ items: PracticeTask[] }>(`${base(role, assignmentId)}/practice-tasks?status=${status}`, { signal });

export const fetchPracticeSessions = (role: PersonaRole, assignmentId: string, page: number, signal?: AbortSignal) =>
  apiRequest<{ items: PracticeSession[]; page: number; per_page: number; total: number }>(`${base(role, assignmentId)}/practice-sessions?page=${page}`, { signal });

export const fetchPracticeSummary = (role: PersonaRole, assignmentId: string, signal?: AbortSignal) =>
  apiRequest<AssignmentPracticeSummary>(`${base(role, assignmentId)}/practice-summary`, { signal });

export const fetchDaySummaries = (date: string, signal?: AbortSignal) =>
  apiRequest<{ items: LessonPracticeSummary[] }>(`/api/teachers/practice-summaries?date=${part(date)}`, { signal });

export const savePracticePlan = (assignmentId: string, plan: PracticePlan) =>
  apiRequest<{ items: PracticeTask[] }>(`${base("teacher", assignmentId)}/practice-plan`, { method: "PUT", body: plan });

export const updatePracticeTask = (taskId: string, body: { status: TaskStatus }) =>
  apiRequest<PracticeTask>(`/api/teachers/practice-tasks/${part(taskId)}`, { method: "PATCH", body });

export const createPracticeSession = (assignmentId: string, draft: SessionDraft) =>
  apiRequest<PracticeSession>(`${base("learner", assignmentId)}/practice-sessions`, { method: "POST", body: draft });

export const deletePracticeSession = (sessionId: string) =>
  apiRequest<void>(`/api/learners/practice-sessions/${part(sessionId)}`, { method: "DELETE" });
