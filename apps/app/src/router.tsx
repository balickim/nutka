// Defines persona-isolated route spaces and guards each protected panel with its matching persona session query.

import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from "@tanstack/react-router";

import type { PersonaRole } from "./api/scheduling";
import { getPersonaRedirect } from "./auth/redirect";
import { ensurePersonaSession } from "./auth/session";
import { updateDocumentTitle } from "./document/title";
import { LoginView } from "./views/login-view";
import { HomeView } from "./views/home-view";
import { PersonaEntryView } from "./views/persona-entry-view";
import { BillingView } from "./views/teacher/billing-view";
import { StudentsView } from "./views/teacher/students-view";
import { StudentView } from "./views/teacher/student-view";
import { TodayView } from "./views/teacher/today-view";
import { AvailabilityView } from "./views/teacher/availability-view";
import { TeacherCalendarView } from "./views/teacher/teacher-calendar-view";

const personaHome: Record<PersonaRole, string> = { teacher: "/teachers", learner: "/learners/calendar" };
const personaLogin: Record<PersonaRole, string> = { teacher: "/teachers/login", learner: "/learners/login" };

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
  beforeLoad: () => skipAuthenticatedPersona("learner"),
  component: () => <LoginView realm="learner" />,
});
const learnerCalendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/calendar",
  beforeLoad: ({ location }) => requirePersona("learner", location.href),
  component: HomeView,
});
const teacherLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/login",
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
