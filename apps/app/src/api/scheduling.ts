// Defines scheduling DTOs, endpoint functions, stable Polish errors, and UTC localization boundaries for scheduling panels.

import { ApiRequestError, apiRequest } from "./transport";

export {
  formatUtcInstant,
  isUtcInstant,
  localizeUtcInstant,
  parseUtcInstant,
} from "../time/utc";

export type PersonaRole = "teacher" | "learner";
export type LessonStatus = "scheduled" | "cancelled";
export type ExceptionKind = "available" | "unavailable";

export type Assignment = {
  id: string;
  teacher: string;
  learner: string;
  teacher_name?: string;
  learner_name?: string;
  active: boolean;
  default_duration_minutes: number;
};

export function assignmentDisplayName(assignment: Assignment, role: PersonaRole): string {
  const name = role === "teacher" ? assignment.learner_name : assignment.teacher_name;
  if (name?.trim()) return name.trim();
  const opaqueId = role === "teacher" ? assignment.learner : assignment.teacher;
  return opaqueId.length > 12 ? `${opaqueId.slice(0, 7)}…${opaqueId.slice(-4)}` : opaqueId;
}

export type AvailabilityRule = {
  id: string;
  teacher: string;
  weekday: number;
  start_time: string;
  end_time: string;
  enabled: boolean;
};

export type AvailabilityException = {
  id: string;
  teacher: string;
  start_at: string;
  end_at: string;
  kind: ExceptionKind;
  note?: string;
};

export type ProtectedInterval = { start_at: string; end_at: string };
export type Lesson = {
  id: string;
  teacher: string;
  learner: string;
  assignment: string;
  start_at: string;
  end_at: string;
  duration_minutes: number;
  status: LessonStatus;
  protected_interval?: ProtectedInterval;
  cancellation_initiator_role?: PersonaRole;
  cancellation_initiator_id?: string;
  cancelled_at?: string;
};

export type Slot = {
  start_at: string;
  end_at: string;
  duration_minutes: number;
  protected_interval?: ProtectedInterval;
};

export type CancellationCounters = { teacher: number; learner: number };
export function ownCancellationCount(counters: CancellationCounters, role: PersonaRole): number {
  return counters[role];
}
export type CalendarResponse = {
  assignments: Assignment[];
  availability_rules: AvailabilityRule[];
  availability_exceptions: AvailabilityException[];
  lessons: Lesson[];
  cancellation_counters: CancellationCounters;
};
export type SlotResponse = { teacher: string; assignment: string; slots: Slot[] };
export type BookingRequest = { start_at: string };
export type LearnerRescheduleRequest = { start_at: string };
export type TeacherRescheduleRequest = { start_at?: string; duration_minutes?: number };
export type RescheduleRequest = LearnerRescheduleRequest | TeacherRescheduleRequest;
export type CancellationRequest = Record<string, never>;
const schedulingErrorCopy: Record<string, string> = {
  unauthenticated: "Zaloguj się ponownie, aby kontynuować.",
  unauthorized: "Nie masz dostępu do tej operacji.",
  missing_intent: "Nie udało się potwierdzić tej operacji. Spróbuj ponownie.",
  invalid_duration: "Czas trwania musi być dodatnią wielokrotnością 15 minut.",
  invalid_grid: "Wybierz termin na granicy 15 minut.",
  invalid_request: "Sprawdź wprowadzone dane.",
  horizon: "Wybierz termin w ciągu najbliższych 14 dni.",
  conflict: "Wybrany termin koliduje z inną lekcją.",
  internal_error: "Nie udało się wykonać operacji.",
  request_failed: "Nie udało się wykonać operacji.",
};

export function getSchedulingErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) return schedulingErrorCopy[error.code] || "Nie udało się wykonać operacji.";
  return "Nie udało się wykonać operacji.";
}

export function fetchTeacherCalendar(signal?: AbortSignal) {
  return apiRequest<CalendarResponse>("/api/teachers/calendar", { signal });
}
export function fetchLearnerCalendar(signal?: AbortSignal) {
  return apiRequest<CalendarResponse>("/api/learners/calendar", { signal });
}
export function fetchLearnerSlots(assignmentId: string, signal?: AbortSignal) {
  return apiRequest<SlotResponse>(`/api/learners/assignments/${encodeURIComponent(assignmentId)}/slots`, { signal });
}
export function updateTeacherAssignment(id: string, body: { active?: boolean; default_duration_minutes?: number }) {
  return apiRequest<Assignment>(`/api/teachers/assignments/${encodeURIComponent(id)}`, { method: "PATCH", body });
}
export function createAvailabilityRule(body: Omit<AvailabilityRule, "id" | "teacher">) {
  return apiRequest<AvailabilityRule>("/api/teachers/availability/rules", { method: "POST", body });
}
export function updateAvailabilityRule(id: string, body: Partial<Omit<AvailabilityRule, "id" | "teacher">>) {
  return apiRequest<AvailabilityRule>(`/api/teachers/availability/rules/${encodeURIComponent(id)}`, { method: "PATCH", body });
}
export function deleteAvailabilityRule(id: string) {
  return apiRequest<void>(`/api/teachers/availability/rules/${encodeURIComponent(id)}`, { method: "DELETE" });
}
export function createAvailabilityException(body: { start_at: string; end_at: string; kind: ExceptionKind; note?: string }) {
  return apiRequest<AvailabilityException>("/api/teachers/availability/exceptions", { method: "POST", body });
}
export function updateAvailabilityException(id: string, body: Partial<{ start_at: string; end_at: string; kind: ExceptionKind; note: string }>) {
  return apiRequest<AvailabilityException>(`/api/teachers/availability/exceptions/${encodeURIComponent(id)}`, { method: "PATCH", body });
}
export function deleteAvailabilityException(id: string) {
  return apiRequest<void>(`/api/teachers/availability/exceptions/${encodeURIComponent(id)}`, { method: "DELETE" });
}
export function bookLesson(assignmentId: string, body: BookingRequest) {
  return apiRequest<Lesson>(`/api/learners/assignments/${encodeURIComponent(assignmentId)}/book`, { method: "POST", body });
}
export function rescheduleLesson(role: PersonaRole, id: string, body: RescheduleRequest) {
  return apiRequest<Lesson>(`/api/${personaSegment(role)}/lessons/${encodeURIComponent(id)}/reschedule`, { method: "PATCH", body });
}
export function cancelLesson(role: PersonaRole, id: string) {
  return apiRequest<Lesson>(`/api/${personaSegment(role)}/lessons/${encodeURIComponent(id)}/cancel`, { method: "POST", body: {} });
}

function personaSegment(role: PersonaRole): string {
  return role === "teacher" ? "teachers" : "learners";
}
