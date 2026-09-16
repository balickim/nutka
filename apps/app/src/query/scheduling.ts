// Declares account-scoped calendar and slot queries plus the scheduling mutations that apply the central cache rules.

import { queryOptions, useMutation, useQueries, useQueryClient } from "@tanstack/react-query";

import { fetchLearnerCalendar, fetchLearnerSlots, fetchTeacherCalendar, type Assignment, type Slot } from "../api/scheduling";
import { applyCacheEffect } from "./effects";
import { queryKeys, queryRules } from "./keys";

type Write = () => Promise<unknown>;

export function teacherCalendarQuery(teacherId: string | undefined) {
  return queryOptions({
    queryKey: queryKeys.calendar("teacher", teacherId ?? ""),
    queryFn: ({ signal }) => fetchTeacherCalendar(signal),
    enabled: Boolean(teacherId),
  });
}

export function learnerCalendarQuery(learnerId: string | undefined) {
  return queryOptions({
    queryKey: queryKeys.calendar("learner", learnerId ?? ""),
    queryFn: ({ signal }) => fetchLearnerCalendar(signal),
    enabled: Boolean(learnerId),
  });
}

export function learnerSlotsQuery(learnerId: string | undefined, assignmentId: string) {
  return queryOptions({
    queryKey: queryKeys.learnerSlots(learnerId ?? "", assignmentId),
    queryFn: ({ signal }) => fetchLearnerSlots(assignmentId, signal),
    enabled: Boolean(learnerId),
  });
}

// One learner slot query runs per active assignment, so the panel reads them as one pending, error, and data set.
export function useLearnerSlots(learnerId: string | undefined, assignments: Assignment[]) {
  const queries = useQueries({ queries: assignments.map((assignment) => learnerSlotsQuery(learnerId, assignment.id)) });
  const slots = new Map<string, Slot[]>(assignments.map((assignment, index) => [assignment.id, queries[index]?.data?.slots || []]));
  return {
    slotsFor: (assignmentId: string) => slots.get(assignmentId) || [],
    pending: queries.some((query) => query.isPending),
    error: queries.find((query) => query.error)?.error,
    refetch: () => queries.forEach((query) => void query.refetch()),
  };
}

// A rejected write already renders through the mutation error, so the caller ignores the rejection.
export function ignoreWriteRejection(write: Promise<unknown>): Promise<void> {
  return write.then(() => undefined, () => undefined);
}

export function useAssignmentWrite() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ write }: { assignmentId: string; write: Write }) => write(),
    onSuccess: (_result, { assignmentId }) => applyCacheEffect(client, queryRules.assignmentWrite(assignmentId)),
  });
}

export function useLessonWrite() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (write: Write) => write(),
    onSuccess: () => applyCacheEffect(client, queryRules.lessonWrite()),
  });
}
