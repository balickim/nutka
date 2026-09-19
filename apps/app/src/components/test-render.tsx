// Renders a component under the providers the application mounts, so a test exercises the same context as the panel.

import { QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { createAppQueryClient } from "../query/client";
import { ToastProvider } from "./Toast";

export function withAppProviders(children: ReactNode) {
  return <QueryClientProvider client={createAppQueryClient()}><ToastProvider>{children}</ToastProvider></QueryClientProvider>;
}
