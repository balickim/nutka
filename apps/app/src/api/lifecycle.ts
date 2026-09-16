// Provides plan-aware booking, lesson lifecycle, outcome, and correction endpoints through the shared transport.
import { apiRequest } from "./transport";
import type {
  Lesson,
  LessonPayment,
  PersonaRole,
  SlotResponse,
} from "./contracts";
const part = (value: string) => encodeURIComponent(value);
const realm = (role: PersonaRole) =>
  role === "teacher" ? "teachers" : "learners";

export type LearnerBookingRequest = { start_at: string };
export type TeacherBookingRequest = {
  start_at: string;
  confirm_short_notice?: boolean;
};
export function bookFlexibleLesson(
  role: "learner",
  assignmentId: string,
  body: LearnerBookingRequest,
): Promise<Lesson>;
export function bookFlexibleLesson(
  role: "teacher",
  assignmentId: string,
  body: TeacherBookingRequest,
): Promise<Lesson>;
export function bookFlexibleLesson(
  role: PersonaRole,
  assignmentId: string,
  body: LearnerBookingRequest | TeacherBookingRequest,
) {
  return apiRequest<Lesson>(
    `/api/${realm(role)}/assignments/${part(assignmentId)}/book`,
    { method: "POST", body },
  );
}
export const fetchSlots = (assignmentId: string, signal?: AbortSignal) =>
  apiRequest<SlotResponse>(
    `/api/learners/assignments/${part(assignmentId)}/slots`,
    { signal },
  );
export const rescheduleCommercialLesson = (
  role: PersonaRole,
  lessonId: string,
  body: { start_at: string },
) =>
  apiRequest<Lesson>(
    `/api/${realm(role)}/lessons/${part(lessonId)}/reschedule`,
    { method: "POST", body },
  );
export const cancelCommercialLesson = (role: PersonaRole, lessonId: string) =>
  apiRequest<Lesson>(`/api/${realm(role)}/lessons/${part(lessonId)}/cancel`, {
    method: "POST",
    body: {},
  });
export const recordLessonOutcome = (
  lessonId: string,
  body: { outcome: "completed" | "learner_no_show" },
) =>
  apiRequest<Lesson>(`/api/teachers/lessons/${part(lessonId)}/outcome`, {
    method: "POST",
    body,
  });
export const recordSettlement = (
  lessonId: string,
  body: { settlement: "paid" | "intentionally_unpaid" },
) =>
  apiRequest<LessonPayment>(`/api/teachers/lessons/${part(lessonId)}/settlement`, {
    method: "POST",
    body,
  });
export const correctSettlement = (
  lessonId: string,
  body: {
    correction: "restore_entitlement" | "restore_allowance" | "restore_settlement";
    reason: string;
  },
) =>
  apiRequest<Lesson>(`/api/teachers/lessons/${part(lessonId)}/correction`, {
    method: "POST",
    body,
  });
