// Mounts the one application QueryClient above the persona router so guards and components share one server-state cache.

import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";

import { queryClient } from "./query/client";
import { router } from "./router";
import "./styles.css";

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
