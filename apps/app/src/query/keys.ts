// Holds the single authoritative registry of frontend query keys and the cache effect of every mutation.

import type { QueryFilters } from "@tanstack/react-query";

import type { PersonaRole } from "../api/scheduling";

export type CacheEffect = {
  cancel: QueryFilters[];
  invalidate: QueryFilters[];
  remove: QueryFilters[];
};

const root = "nutka";

export const queryKeys = {
  persona: (role: PersonaRole) => [root, role] as const,
  session: (role: PersonaRole) => [root, role, "session"] as const,
  policy: (role: PersonaRole) => [root, role, "policy"] as const,
  calendars: (role: PersonaRole) => [root, role, "calendar"] as const,
  calendar: (role: PersonaRole, accountId: string) =>
    [root, role, "calendar", accountId] as const,
  learnerSlotsRoot: () => [root, "learner", "slots"] as const,
  learnerSlots: (learnerId: string, assignmentId: string) =>
    [root, "learner", "slots", learnerId, assignmentId] as const,
  commercialSummaries: (role: PersonaRole) =>
    [root, role, "assignment-summary"] as const,
  commercialSummary: (
    role: PersonaRole,
    accountId: string,
    assignmentId: string,
  ) => [root, role, "assignment-summary", accountId, assignmentId] as const,
  // Packages and contracts stay under the summary prefix so one assignment rule invalidates every commercial read of that assignment.
  assignmentPackages: (
    role: PersonaRole,
    accountId: string,
    assignmentId: string,
  ) =>
    [root, role, "assignment-summary", accountId, assignmentId, "packages"] as const,
  assignmentContracts: (
    role: PersonaRole,
    accountId: string,
    assignmentId: string,
  ) =>
    [root, role, "assignment-summary", accountId, assignmentId, "contracts"] as const,
  // The payment-due read stays under the summary prefix so every assignment commercial rule also refreshes it.
  paymentDue: (accountId: string, assignmentId: string) =>
    [root, "learner", "assignment-summary", accountId, assignmentId, "payment-due"] as const,
  paymentDetails: (accountId: string) =>
    [root, "teacher", "payment-details", accountId] as const,
  contractSeriesRoot: (role: PersonaRole) =>
    [root, role, "contract-series"] as const,
  contractSeries: (
    role: PersonaRole,
    assignmentId: string,
    contractId: string,
  ) => [root, role, "contract-series", assignmentId, contractId] as const,
  contractMonths: (
    role: PersonaRole,
    assignmentId: string,
    contractId: string,
  ) =>
    [root, role, "contract-series", assignmentId, contractId, "months"] as const,
  financialWorkRoot: (role: PersonaRole) =>
    [root, role, "financial-work"] as const,
  financialWork: (role: PersonaRole, accountId: string, assignmentId = "all") =>
    [root, role, "financial-work", accountId, assignmentId] as const,
  historyRoot: (role: PersonaRole) => [root, role, "history"] as const,
  history: (role: PersonaRole, accountId: string, assignmentId: string) =>
    [root, role, "history", accountId, assignmentId] as const,
  materialsRoot: (role: PersonaRole) => [root, role, "materials"] as const,
  materials: (role: PersonaRole, accountId: string, assignmentId: string) =>
    [root, role, "materials", accountId, assignmentId] as const,
  piecesRoot: (role: PersonaRole) => [root, role, "pieces"] as const,
  pieces: (role: PersonaRole, accountId: string, assignmentId: string) =>
    [root, role, "pieces", accountId, assignmentId] as const,
  practiceRoot: (role: PersonaRole) => [root, role, "practice"] as const,
  practiceTasks: (role: PersonaRole, accountId: string, assignmentId: string, status: string) =>
    [root, role, "practice", accountId, assignmentId, "tasks", status] as const,
  practiceSessions: (role: PersonaRole, accountId: string, assignmentId: string, page: number) =>
    [root, role, "practice", accountId, assignmentId, "sessions", page] as const,
  practiceSummary: (role: PersonaRole, accountId: string, assignmentId: string) =>
    [root, role, "practice", accountId, assignmentId, "summary"] as const,
  practiceDaysRoot: () => [root, "teacher", "practice-day"] as const,
  practiceDay: (accountId: string, date: string) =>
    [root, "teacher", "practice-day", accountId, date] as const,
  lessonNotesRoot: (role: PersonaRole) => [root, role, "lesson-notes"] as const,
  lessonNotes: (role: PersonaRole, accountId: string, assignmentId: string) =>
    [root, role, "lesson-notes", accountId, assignmentId] as const,
  unresolvedWork: (role: "teacher", accountId: string) =>
    [root, role, "unresolved-work", accountId] as const,
};

const effect = (parts: Partial<CacheEffect>): CacheEffect => ({
  cancel: [],
  invalidate: [],
  remove: [],
  ...parts,
});
const bothCalendars = (): QueryFilters[] => [
  { queryKey: queryKeys.calendars("teacher") },
  { queryKey: queryKeys.calendars("learner") },
];
const everyLearnerSlot = (): QueryFilters => ({
  queryKey: queryKeys.learnerSlotsRoot(),
});
const assignmentSlots = (assignmentId: string): QueryFilters => ({
  queryKey: queryKeys.learnerSlotsRoot(),
  predicate: (query) => query.queryKey[4] === assignmentId,
});
const affectedSummaries = (assignmentId: string): QueryFilters[] =>
  ["teacher" as PersonaRole, "learner" as PersonaRole].map((role) => ({
    queryKey: queryKeys.commercialSummaries(role),
    predicate: (query) => query.queryKey[4] === assignmentId,
  }));
const allFinancialWork = (): QueryFilters[] => [
  { queryKey: queryKeys.financialWorkRoot("teacher") },
  { queryKey: queryKeys.financialWorkRoot("learner") },
];
const allHistory = (): QueryFilters[] => [
  { queryKey: queryKeys.historyRoot("teacher") },
  { queryKey: queryKeys.historyRoot("learner") },
];
const teacherUnresolvedWork = (): QueryFilters[] => [
  { queryKey: [root, "teacher", "unresolved-work"] },
];
const allContractSeries = (): QueryFilters[] => [
  { queryKey: queryKeys.contractSeriesRoot("teacher") },
  { queryKey: queryKeys.contractSeriesRoot("learner") },
];

const personas: PersonaRole[] = ["teacher", "learner"];
const assignmentMaterials = (assignmentId: string): QueryFilters[] =>
  personas.map((role) => ({
    queryKey: queryKeys.materialsRoot(role),
    predicate: (query) => query.queryKey[4] === assignmentId,
  }));
const assignmentPieces = (assignmentId: string): QueryFilters[] =>
  personas.map((role) => ({
    queryKey: queryKeys.piecesRoot(role),
    predicate: (query) => query.queryKey[4] === assignmentId,
  }));
const assignmentPractice = (assignmentId: string): QueryFilters[] =>
  personas.map((role) => ({
    queryKey: queryKeys.practiceRoot(role),
    predicate: (query) => query.queryKey[4] === assignmentId,
  }));

const commercialReadEffects = (assignmentId?: string): QueryFilters[] => [
  ...(assignmentId
    ? affectedSummaries(assignmentId)
    : [
        { queryKey: queryKeys.commercialSummaries("teacher") },
        { queryKey: queryKeys.commercialSummaries("learner") },
      ]),
  ...allFinancialWork(),
  ...allHistory(),
  ...teacherUnresolvedWork(),
];
const lessonMutationEffects = (assignmentId?: string): CacheEffect =>
  effect({
    invalidate: [
      ...bothCalendars(),
      everyLearnerSlot(),
      ...commercialReadEffects(assignmentId),
    ],
  });

export const queryRules = {
  // An activity or duration change alters only the slots of that assignment.
  assignmentWrite: (assignmentId: string): CacheEffect =>
    effect({
      invalidate: [
        ...bothCalendars(),
        assignmentSlots(assignmentId),
        ...affectedSummaries(assignmentId),
      ],
    }),
  // A participant conflict from one lesson can block instants across assignments.
  lessonWrite: (): CacheEffect => lessonMutationEffects(),
  bookingWrite: (assignmentId?: string): CacheEffect =>
    lessonMutationEffects(assignmentId),
  lifecycleWrite: (assignmentId?: string): CacheEffect =>
    lessonMutationEffects(assignmentId),
  planWrite: (assignmentId?: string): CacheEffect =>
    effect({
      invalidate: [
        ...lessonMutationEffects(assignmentId).invalidate,
        ...allContractSeries(),
      ],
    }),
  availabilityCommit: (assignmentId?: string): CacheEffect =>
    effect({
      invalidate: [
        ...bothCalendars(),
        everyLearnerSlot(),
        ...commercialReadEffects(assignmentId),
        ...allContractSeries(),
      ],
    }),
  outcomeWrite: (assignmentId?: string): CacheEffect =>
    lessonMutationEffects(assignmentId),
  settlementWrite: (assignmentId?: string): CacheEffect =>
    lessonMutationEffects(assignmentId),
  correctionWrite: (assignmentId?: string): CacheEffect =>
    effect({
      invalidate: [
        ...lessonMutationEffects(assignmentId).invalidate,
        ...allContractSeries(),
      ],
    }),
  // A material write changes the materials of one assignment and the material counters of its pieces, for both personas.
  materialWrite: (assignmentId: string): CacheEffect =>
    effect({
      invalidate: [
        ...assignmentMaterials(assignmentId),
        ...assignmentPieces(assignmentId),
      ],
    }),
  // A piece delete clears material links, so a piece write also refreshes the materials of that assignment.
  pieceWrite: (assignmentId: string): CacheEffect =>
    effect({
      invalidate: [
        ...assignmentPieces(assignmentId),
        ...assignmentMaterials(assignmentId),
      ],
    }),
  // A lesson note write changes only the note lists of one assignment, for both personas.
  noteWrite: (assignmentId: string): CacheEffect =>
    effect({
      invalidate: (["teacher", "learner"] as PersonaRole[]).map((role) => ({
        queryKey: queryKeys.lessonNotesRoot(role),
        predicate: (query) => query.queryKey[4] === assignmentId,
      })),
    }),
  // A task or session write changes the tasks, sessions, and summaries of one assignment and any teacher day summary.
  practiceWrite: (assignmentId: string): CacheEffect =>
    effect({
      invalidate: [
        ...assignmentPractice(assignmentId),
        { queryKey: queryKeys.practiceDaysRoot() },
      ],
    }),
  // Transfer details change only the teacher's own details read. Learners read them in their own session.
  paymentDetailsWrite: (accountId: string): CacheEffect =>
    effect({ invalidate: [{ queryKey: queryKeys.paymentDetails(accountId) }] }),
  // Logout, expiry, cross-tab logout, a session 401, and account replacement all end the right to read that persona cache.
  // The session entry survives removal because its caller writes the replacing session value into it.
  personaCleared: (role: PersonaRole): CacheEffect =>
    effect({
      cancel: [{ queryKey: queryKeys.persona(role) }],
      remove: [
        {
          queryKey: queryKeys.persona(role),
          predicate: (query) => query.queryKey[2] !== "session",
        },
      ],
    }),
};
