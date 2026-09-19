// Defines scheduling DTOs, endpoint functions, stable Polish errors, and UTC localization boundaries for scheduling panels.

import { ApiRequestError, apiRequest } from "./transport";
import { polishError } from "./copy";
import type {
  Assignment,
  AvailabilityException,
  AvailabilityRule,
  CalendarResponse,
  ExceptionKind,
  Lesson,
  PersonaRole,
  Slot,
} from "./contracts";

export {
  formatUtcInstant,
  isUtcInstant,
  localizeUtcInstant,
  parseUtcInstant,
} from "../time/utc";

export type LessonStatus = "scheduled" | "cancelled" | "omitted";
export type {
  Assignment,
  AvailabilityException,
  AvailabilityRule,
  CalendarResponse,
  ExceptionKind,
  Lesson,
  PersonaRole,
  Slot,
} from "./contracts";

export function assignmentDisplayName(
  assignment: Assignment,
  role: PersonaRole,
): string {
  const name =
    role === "teacher" ? assignment.learner_name : assignment.teacher_name;
  if (name?.trim()) return name.trim();
  return opaqueIdLabel(role === "teacher" ? assignment.learner : assignment.teacher);
}

export function opaqueIdLabel(id: string): string {
  return id.length > 12 ? `${id.slice(0, 7)}…${id.slice(-4)}` : id;
}

export type SlotResponse = {
  teacher: string;
  assignment: string;
  slots: Slot[];
};
export type BookingRequest = { start_at: string };
export type LearnerRescheduleRequest = { start_at: string };
export type RescheduleRequest = LearnerRescheduleRequest;
export type CancellationRequest = Record<string, never>;
export function getSchedulingErrorMessage(error: unknown): string {
  return error instanceof ApiRequestError ? polishError(error.code) : polishError("request_failed");
}

export function fetchTeacherCalendar(signal?: AbortSignal) {
  return apiRequest<CalendarResponse>("/api/teachers/calendar", { signal });
}
export function fetchLearnerCalendar(signal?: AbortSignal) {
  return apiRequest<CalendarResponse>("/api/learners/calendar", { signal });
}
export function fetchLearnerSlots(assignmentId: string, signal?: AbortSignal) {
  return apiRequest<SlotResponse>(
    `/api/learners/assignments/${encodeURIComponent(assignmentId)}/slots`,
    { signal },
  );
}
export function bookLesson(assignmentId: string, body: BookingRequest) {
  return apiRequest<Lesson>(
    `/api/learners/assignments/${encodeURIComponent(assignmentId)}/book`,
    { method: "POST", body },
  );
}
export function rescheduleLesson(
  role: PersonaRole,
  id: string,
  body: RescheduleRequest,
) {
  return apiRequest<Lesson>(
    `/api/${personaSegment(role)}/lessons/${encodeURIComponent(id)}/reschedule`,
    { method: "POST", body: { start_at: body.start_at } },
  );
}
export function cancelLesson(role: PersonaRole, id: string) {
  return apiRequest<Lesson>(
    `/api/${personaSegment(role)}/lessons/${encodeURIComponent(id)}/cancel`,
    { method: "POST", body: {} },
  );
}

function personaSegment(role: PersonaRole): string {
  return role === "teacher" ? "teachers" : "learners";
}
