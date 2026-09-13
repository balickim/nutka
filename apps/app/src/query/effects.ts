// Applies a registry cache effect to a QueryClient so callers never build their own cancellation, invalidation, or removal filters.

import type { QueryClient } from "@tanstack/react-query";

import type { CacheEffect } from "./keys";

export async function applyCacheEffect(client: QueryClient, effect: CacheEffect): Promise<void> {
  await Promise.all(effect.cancel.map((filters) => client.cancelQueries(filters)));
  effect.remove.forEach((filters) => client.removeQueries(filters));
  await Promise.all(effect.invalidate.map((filters) => client.invalidateQueries(filters)));
}
