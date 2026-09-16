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
import { LoginView } from "./views/LoginView";
import { HomeView } from "./views/HomeView";
import { PersonaEntryView } from "./views/PersonaEntryView";
import { TeacherView } from "./views/TeacherView";

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
  component: TeacherView,
});
const teacherAvailabilityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/availability",
  beforeLoad: ({ location }) => requirePersona("teacher", location.href),
  component: () => <TeacherView availability />,
});
const entryRoute = createRoute({ getParentRoute: () => rootRoute, path: "/", component: PersonaEntryView });
const routeTree = rootRoute.addChildren([
  entryRoute,
  learnerLoginRoute,
  learnerCalendarRoute,
  teacherLoginRoute,
  teacherRoute,
  teacherAvailabilityRoute,
]);

export const router = createRouter({ routeTree, defaultPreload: "intent" });
updateDocumentTitle(router.state.location.pathname);
router.subscribe("onResolved", () => updateDocumentTitle(router.state.location.pathname));

declare module "@tanstack/react-router" {
  interface Register { router: typeof router; }
}
