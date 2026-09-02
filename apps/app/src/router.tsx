import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from "@tanstack/react-router";

import { bootstrapAuth } from "./auth/auth";
import { HomeView } from "./views/HomeView";
import { LoginView } from "./views/LoginView";

const rootRoute = createRootRoute({ component: () => <Outlet /> });
const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  beforeLoad: async () => {
    const result = await bootstrapAuth();
    if (result.kind === "authenticated") throw redirect({ to: "/" });
  },
  component: LoginView,
});
const homeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  beforeLoad: async ({ location }) => {
    const result = await bootstrapAuth();
    if (result.kind === "unauthenticated") {
      throw redirect({
        to: "/login",
        search: { redirect: location.href },
      });
    }
  },
  component: HomeView,
});
const routeTree = rootRoute.addChildren([loginRoute, homeRoute]);
export const router = createRouter({ routeTree, defaultPreload: "intent" });

declare module "@tanstack/react-router" {
  interface Register { router: typeof router; }
}
