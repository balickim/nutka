// Creates the one application QueryClient that route guards, hooks, and lifecycle effects share.

import { QueryClient } from "@tanstack/react-query";

import { isRetryableFailure } from "../api/transport";

export function createAppQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: (failureCount, error) => failureCount < 1 && isRetryableFailure(error),
        refetchOnWindowFocus: false,
      },
      mutations: { retry: false },
    },
  });
}

export const queryClient = createAppQueryClient();
