// Defines persona-isolated route spaces and guards each protected panel with its matching session store.

import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from "@tanstack/react-router";

import { bootstrapAuth as bootstrapLearner } from "./auth/auth";
import { bootstrapAuth as bootstrapTeacher } from "./auth/teacher";
import { getPersonaRedirect } from "./auth/redirect";
import { updateDocumentTitle } from "./document/title";
import { LoginView } from "./views/LoginView";
import { HomeView } from "./views/HomeView";
import { PersonaEntryView } from "./views/PersonaEntryView";
import { TeacherView } from "./views/TeacherView";

const rootRoute = createRootRoute({ component: () => <Outlet /> });
const learnerLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/login",
  beforeLoad: async () => {
    const result = await bootstrapLearner();
    if (result.kind === "authenticated") throw redirect({ to: "/learners/calendar" });
  },
  component: () => <LoginView realm="learner" />,
});
const learnerCalendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/learners/calendar",
  beforeLoad: async ({ location }) => {
    const result = await bootstrapLearner();
    if (result.kind === "unauthenticated") {
      throw redirect({
        to: "/learners/login",
        search: { redirect: getPersonaRedirect(location.href, "learner") },
      });
    }
  },
  component: HomeView,
});
const teacherLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/login",
  beforeLoad: async () => {
    const result = await bootstrapTeacher();
    if (result.kind === "authenticated") throw redirect({ to: "/teachers" });
  },
  component: () => <LoginView realm="teacher" />,
});
const teacherRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers",
  beforeLoad: async ({ location }) => {
    const result = await bootstrapTeacher();
    if (result.kind === "unauthenticated") {
      throw redirect({
        to: "/teachers/login",
        search: { redirect: getPersonaRedirect(location.href, "teacher") },
      });
    }
  },
  component: TeacherView,
});
const teacherAvailabilityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/teachers/availability",
  beforeLoad: async ({ location }) => {
    const result = await bootstrapTeacher();
    if (result.kind === "unauthenticated") {
      throw redirect({
        to: "/teachers/login",
        search: { redirect: getPersonaRedirect(location.href, "teacher") },
      });
    }
  },
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
