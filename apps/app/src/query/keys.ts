// Holds the single authoritative registry of frontend query keys and the cache effect of every mutation.

import type { QueryFilters } from "@tanstack/react-query";

import type { PersonaRole } from "../api/scheduling";

export type CacheEffect = { cancel: QueryFilters[]; invalidate: QueryFilters[]; remove: QueryFilters[] };

const root = "nutka";

export const queryKeys = {
  persona: (role: PersonaRole) => [root, role] as const,
  session: (role: PersonaRole) => [root, role, "session"] as const,
  calendars: (role: PersonaRole) => [root, role, "calendar"] as const,
  calendar: (role: PersonaRole, accountId: string) => [root, role, "calendar", accountId] as const,
  learnerSlotsRoot: () => [root, "learner", "slots"] as const,
  learnerSlots: (learnerId: string, assignmentId: string) => [root, "learner", "slots", learnerId, assignmentId] as const,
};

const effect = (parts: Partial<CacheEffect>): CacheEffect => ({ cancel: [], invalidate: [], remove: [], ...parts });
const bothCalendars = (): QueryFilters[] => [{ queryKey: queryKeys.calendars("teacher") }, { queryKey: queryKeys.calendars("learner") }];
const everyLearnerSlot = (): QueryFilters => ({ queryKey: queryKeys.learnerSlotsRoot() });
const assignmentSlots = (assignmentId: string): QueryFilters => ({
  queryKey: queryKeys.learnerSlotsRoot(),
  predicate: (query) => query.queryKey[4] === assignmentId,
});

export const queryRules = {
  // A rule or exception changes which instants any learner may book.
  availabilityWrite: (): CacheEffect => effect({ invalidate: [...bothCalendars(), everyLearnerSlot()] }),
  // An activity or duration change alters only the slots of that assignment.
  assignmentWrite: (assignmentId: string): CacheEffect => effect({ invalidate: [...bothCalendars(), assignmentSlots(assignmentId)] }),
  // A participant conflict from one lesson can block instants across assignments.
  lessonWrite: (): CacheEffect => effect({ invalidate: [...bothCalendars(), everyLearnerSlot()] }),
  // Logout, expiry, cross-tab logout, a session 401, and account replacement all end the right to read that persona cache.
  // The session entry survives removal because its caller writes the replacing session value into it.
  personaCleared: (role: PersonaRole): CacheEffect =>
    effect({
      cancel: [{ queryKey: queryKeys.persona(role) }],
      remove: [{ queryKey: queryKeys.persona(role), predicate: (query) => query.queryKey[2] !== "session" }],
    }),
};
