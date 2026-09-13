// Models each persona session as query cache data and exposes the login, logout, expiry, and cross-tab effects on that cache.

import { queryOptions, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { z } from "zod";

import { ApiRequestError, apiRequest } from "../api/transport";
import type { PersonaRole } from "../api/scheduling";
import { queryClient } from "../query/client";
import { applyCacheEffect } from "../query/effects";
import { queryKeys, queryRules } from "../query/keys";
import { clearExpiry, publishPersonaEvent, scheduleExpiry, subscribePersonaEvents } from "./lifecycle";

const recordSchema = z.object({
  id: z.string(),
  email: z.string(),
  name: z.string().optional(),
  display_name: z.string().optional(),
  timezone: z.string().optional(),
});

export type PersonaRecord = z.infer<typeof recordSchema>;
export type AuthenticatedSession = { kind: "authenticated"; record: PersonaRecord; expiresAt?: string };
export type SessionResult = AuthenticatedSession | { kind: "unauthenticated" };
export type Credentials = { email: string; password: string };
export type AuthErrorKind = "invalid-credentials" | "retryable";

const personaEndpoints: Record<PersonaRole, { me: string; logout: string; collection: string }> = {
  teacher: { me: "/api/teachers/auth/me", logout: "/api/teachers/auth/logout", collection: "teachers" },
  learner: { me: "/api/learners/auth/me", logout: "/api/learners/auth/logout", collection: "learners" },
};

export function authenticatedRecord(result: SessionResult | undefined): PersonaRecord | undefined {
  return result?.kind === "authenticated" ? result.record : undefined;
}

export function personaDisplayName(record: PersonaRecord): string {
  return record.name || record.display_name || record.email;
}

export function classifyAuthError(error: unknown): AuthErrorKind {
  const status = typeof error === "object" && error !== null && "status" in error ? Number((error as { status?: unknown }).status) : 0;
  return status >= 400 && status < 500 ? "invalid-credentials" : "retryable";
}

function parseSession(body: unknown): AuthenticatedSession {
  const response = (body ?? {}) as { record?: unknown; session_expires_at?: unknown };
  const parsed = recordSchema.safeParse(response.record);
  if (!parsed.success) throw new Error("Invalid persona response");
  return {
    kind: "authenticated",
    record: parsed.data,
    expiresAt: typeof response.session_expires_at === "string" ? response.session_expires_at : undefined,
  };
}

// Every resolved session result owns the expiry timer of its role, so a bootstrap read arms it exactly like a login.
function trackExpiry(role: PersonaRole, result: SessionResult): SessionResult {
  if (result.kind !== "authenticated") clearExpiry(role);
  else scheduleExpiry(role, result.expiresAt, () => void expirePersonaSession(role));
  return result;
}

async function readSession(role: PersonaRole, signal: AbortSignal): Promise<SessionResult> {
  try {
    return trackExpiry(role, parseSession(await apiRequest<unknown>(personaEndpoints[role].me, { signal })));
  } catch (error) {
    if (error instanceof ApiRequestError && error.status === 401) return trackExpiry(role, { kind: "unauthenticated" });
    throw error;
  }
}

// The session survives until a lifecycle effect replaces it, so a guard and its component share one request.
export function sessionQueryOptions(role: PersonaRole) {
  return queryOptions({
    queryKey: queryKeys.session(role),
    queryFn: ({ signal }) => readSession(role, signal),
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

export function ensurePersonaSession(role: PersonaRole): Promise<SessionResult> {
  return queryClient.ensureQueryData(sessionQueryOptions(role));
}

export function usePersonaSession(role: PersonaRole) {
  return useQuery(sessionQueryOptions(role));
}

export async function storePersonaSession(client: QueryClient, role: PersonaRole, result: SessionResult): Promise<void> {
  await applyCacheEffect(client, queryRules.personaCleared(role));
  client.setQueryData(queryKeys.session(role), trackExpiry(role, result));
}

async function expirePersonaSession(role: PersonaRole): Promise<void> {
  await storePersonaSession(queryClient, role, { kind: "unauthenticated" });
  publishPersonaEvent(role, "forced-unauthenticated");
}

subscribePersonaEvents((role, event) => {
  if (event === "login") {
    void queryClient.invalidateQueries({ queryKey: queryKeys.session(role) });
    return;
  }
  void storePersonaSession(queryClient, role, { kind: "unauthenticated" });
});

export function usePersonaLogin(role: PersonaRole) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ email, password }: Credentials) => {
      const body = await apiRequest<unknown>(`/api/collections/${personaEndpoints[role].collection}/auth-with-password`, {
        method: "POST",
        body: { identity: email, password },
      });
      return parseSession(body);
    },
    onSuccess: async (result) => {
      await storePersonaSession(client, role, result);
      publishPersonaEvent(role, "login");
    },
  });
}

export function usePersonaLogout(role: PersonaRole) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      try {
        await apiRequest<void>(personaEndpoints[role].logout, { method: "POST" });
      } catch {
        // The local persona cache still clears when the server cannot be reached.
      }
    },
    onSuccess: async () => {
      await storePersonaSession(client, role, { kind: "unauthenticated" });
      publishPersonaEvent(role, "logout");
    },
  });
}
