// Defines scheduling DTOs, authorized API calls, stable Polish errors, and UTC localization boundaries for scheduling panels.

import { apiUrl } from "./url";

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
export type ApiError = { code: string; message: string };

export class SchedulingApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(error: ApiError, status: number) {
    super(error.message);
    this.name = "SchedulingApiError";
    this.code = error.code;
    this.status = status;
  }
}

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
  if (error instanceof SchedulingApiError) return schedulingErrorCopy[error.code] || "Nie udało się wykonać operacji.";
  return "Nie udało się wykonać operacji.";
}

const intentHeaders = { "X-Requested-With": "fetch" };

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(apiUrl(path), {
    ...options,
    credentials: "include",
    headers: { ...intentHeaders, ...options.headers },
  });
  if (!response.ok) {
    let error: ApiError = { code: "request_failed", message: "Nie udało się wykonać operacji." };
    try {
      const body = (await response.json()) as Partial<ApiError>;
      if (typeof body.code === "string" && typeof body.message === "string") error = body as ApiError;
    } catch { /* Stable generic message for non-JSON responses. */ }
    throw new SchedulingApiError(error, response.status);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

function jsonOptions(method: string, body: unknown): RequestInit {
  return { method, headers: { "Content-Type": "application/json", "X-Requested-With": "fetch" }, body: JSON.stringify(body) };
}

export function fetchTeacherCalendar() { return request<CalendarResponse>("/api/teachers/calendar"); }
export function fetchLearnerCalendar() { return request<CalendarResponse>("/api/learners/calendar"); }
export function updateTeacherAssignment(id: string, body: { active?: boolean; default_duration_minutes?: number }) {
  return request<Assignment>(`/api/teachers/assignments/${encodeURIComponent(id)}`, jsonOptions("PATCH", body));
}
export function createAvailabilityRule(body: Omit<AvailabilityRule, "id" | "teacher">) {
  return request<AvailabilityRule>("/api/teachers/availability/rules", jsonOptions("POST", body));
}
export function updateAvailabilityRule(id: string, body: Partial<Omit<AvailabilityRule, "id" | "teacher">>) {
  return request<AvailabilityRule>(`/api/teachers/availability/rules/${encodeURIComponent(id)}`, jsonOptions("PATCH", body));
}
export function deleteAvailabilityRule(id: string) {
  return request<void>(`/api/teachers/availability/rules/${encodeURIComponent(id)}`, { method: "DELETE", headers: intentHeaders });
}
export function createAvailabilityException(body: { start_at: string; end_at: string; kind: ExceptionKind; note?: string }) {
  return request<AvailabilityException>("/api/teachers/availability/exceptions", jsonOptions("POST", body));
}
export function updateAvailabilityException(id: string, body: Partial<{ start_at: string; end_at: string; kind: ExceptionKind; note: string }>) {
  return request<AvailabilityException>(`/api/teachers/availability/exceptions/${encodeURIComponent(id)}`, jsonOptions("PATCH", body));
}
export function deleteAvailabilityException(id: string) {
  return request<void>(`/api/teachers/availability/exceptions/${encodeURIComponent(id)}`, { method: "DELETE", headers: intentHeaders });
}
export function fetchLearnerSlots(assignmentId: string) {
  return request<SlotResponse>(`/api/learners/assignments/${encodeURIComponent(assignmentId)}/slots`);
}
export function bookLesson(assignmentId: string, body: BookingRequest) {
  return request<Lesson>(`/api/learners/assignments/${encodeURIComponent(assignmentId)}/book`, jsonOptions("POST", body));
}
export function rescheduleLesson(role: PersonaRole, id: string, body: RescheduleRequest) {
  return request<Lesson>(`/api/${role === "teacher" ? "teachers" : "learners"}/lessons/${encodeURIComponent(id)}/reschedule`, jsonOptions("PATCH", body));
}
export function cancelLesson(role: PersonaRole, id: string) {
  return request<Lesson>(`/api/${role === "teacher" ? "teachers" : "learners"}/lessons/${encodeURIComponent(id)}/cancel`, jsonOptions("POST", {}));
}
