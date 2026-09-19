// Defines persona-isolated route spaces and guards each protected panel with its matching persona session query.

import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from "@tanstack/react-router";

import type { PersonaRole } from "./api/scheduling";
import { getPersonaRedirect, personaHome } from "./auth/redirect";
import { ensurePersonaSession } from "./auth/session";
import { updateDocumentTitle } from "./document/title";
import { LoginView } from "./views/login-view";
import { LessonsView } from "./views/learner/lessons-view";
import { PiecesView } from "./views/learner/pieces-view";
import { StartView } from "./views/learner/start-view";
import { PersonaEntryView } from "./views/persona-entry-view";
import { BillingView } from "./views/teacher/billing-view";
import { StudentsView } from "./views/teacher/students-view";
import { StudentView } from "./views/teacher/student-view";
import { TodayView } from "./views/teacher/today-view";
import { AvailabilityView } from "./views/teacher/availability-view";
import { TeacherCalendarView } from "./views/teacher/teacher-calendar-view";

const personaLogin = { teacher: "/teachers/login", learner: "/learners/login" } as const satisfies Record<PersonaRole, string>;
// Login routes keep the post-login target that the persona guard sets.
const loginSearch = (search: Record<string, unknown>): { redirect?: string } => (typeof search.redirect === "string" ? { redirect: search.redirect } : {});

// A retryable session failure keeps the route so the panel can render its own retry control.
async function readSession(role: PersonaRole) {
  return ensurePersonaSession(role).catch(() => null);
}

async function requirePersona(role: PersonaRole, href: string) {
  const result = await readSession(role);
  if (result?.kind !== "unauthenticated") return;
  throw redirect({ to: personaLogin[role], search: { redirect: getPersonaRedirect(href, role) } });
}

async function skipAuthenticatedPersona(role: PersonaRole) {
  const result = await readSession(role);
  if (result?.kind === "authenticated") throw redirect({ to: personaHome[role] });
}

const rootRoute = createRootRoute({ component: () => <Outlet /> });
const learnerLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/login",
  validateSearch: loginSearch,
  beforeLoad: () => skipAuthenticatedPersona("learner"),
  component: () => <LoginView realm="learner" />,
});
// The optional `a` parameter selects one learner assignment across every learner screen.
export type LearnerSearch = { a?: string };
const learnerSearch = (search: Record<string, unknown>): LearnerSearch => (typeof search.a === "string" && search.a ? { a: search.a } : {});

const learnerStartRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners",
  validateSearch: learnerSearch,
  beforeLoad: ({ location }) => requirePersona("learner", location.href),
  component: StartView,
});
const learnerLessonsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/lessons",
  validateSearch: learnerSearch,
  beforeLoad: ({ location }) => requirePersona("learner", location.href),
  component: LessonsView,
});
const learnerPiecesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/pieces",
  validateSearch: learnerSearch,
  beforeLoad: ({ location }) => requirePersona("learner", location.href),
  component: PiecesView,
});
// Saved bookmarks of the former single learner panel open the lessons screen.
const learnerCalendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/calendar",
  beforeLoad: () => { throw redirect({ to: "/learners/lessons", replace: true }); },
});
const teacherLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/login",
  validateSearch: loginSearch,
  beforeLoad: () => skipAuthenticatedPersona("teacher"),
  component: () => <LoginView realm="teacher" />,
});
const teacherRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: TodayView,
});
const teacherAvailabilityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/availability",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: AvailabilityView,
});
const teacherCalendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/calendar",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: TeacherCalendarView,
});
const teacherStudentsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/students",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: StudentsView,
});
const teacherStudentRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/students/$assignmentId",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: StudentView,
});
const teacherBillingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/billing",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: BillingView,
});
const entryRoute = createRoute({ getParentRoute: () => rootRoute, path: "/", component: PersonaEntryView });
const routeTree = rootRoute.addChildren([
  entryRoute,
  learnerLoginRoute,
  learnerStartRoute,
  learnerLessonsRoute,
  learnerPiecesRoute,
  learnerCalendarRoute,
  teacherLoginRoute,
  teacherRoute,
  teacherAvailabilityRoute,
  teacherCalendarRoute,
  teacherStudentsRoute,
  teacherStudentRoute,
  teacherBillingRoute,
]);

export const router = createRouter({ routeTree, defaultPreload: "intent" });
updateDocumentTitle(router.state.location.pathname);
router.subscribe("onResolved", () => updateDocumentTitle(router.state.location.pathname));

declare module "@tanstack/react-router" {
  interface Register { router: typeof router; }
}
