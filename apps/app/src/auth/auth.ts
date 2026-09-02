import type { RecordModel } from "pocketbase";
import { z } from "zod";

import { apiUrl, pb } from "../api/pocketbase";

const sessionFallback = 12 * 60 * 60 * 1000;
const learnerSchema = z
  .object({
    id: z.string(),
    email: z.string(),
    name: z.string().optional(),
    display_name: z.string().optional(),
  })
  .passthrough();

export type Learner = z.infer<typeof learnerSchema> & RecordModel;
export type MeResult =
  | { kind: "authenticated"; record: Learner; expiresAt?: string }
  | { kind: "unauthenticated" }
  | { kind: "retryable"; error: Error };
export type AuthState = {
  status: "idle" | "loading" | "ready" | "error";
  record: Learner | null;
  error: Error | null;
};

let state: AuthState = { status: "idle", record: null, error: null };
let bootstrapPromise: Promise<MeResult> | null = null;
let expiryTimer: number | undefined;
let scheduledExpiryAt = 0;
const listeners = new Set<() => void>();
const authChannel = typeof BroadcastChannel === "undefined" ? null : new BroadcastChannel("nutka-auth");

authChannel?.addEventListener("message", (event) => {
  const type = event.data?.type;
  if (type === "logout" || type === "forced-unauthenticated") clearAuthState(false);
  if (type === "login" && !state.record) void bootstrapAuth();
});

if (typeof document !== "undefined") {
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && scheduledExpiryAt > 0 && Date.now() >= scheduledExpiryAt) {
      clearAuthState();
    }
  });
}

export function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
export function getAuthState(): AuthState { return state; }
export function getLearnerDisplayName(record: Learner): string {
  return record.name || record.display_name || record.email;
}
function notify() { listeners.forEach((listener) => listener()); }
function clearExpiry() {
  if (expiryTimer !== undefined) window.clearTimeout(expiryTimer);
  expiryTimer = undefined;
  scheduledExpiryAt = 0;
}
function clearAuthState(broadcast = true) {
  clearExpiry();
  pb.authStore.clear();
  state = { status: "ready", record: null, error: null };
  notify();
  if (broadcast) authChannel?.postMessage({ type: "logout" });
}
function scheduleExpiry(expiresAt?: string) {
  clearExpiry();
  const reported = expiresAt ? Date.parse(expiresAt) : Number.NaN;
  const expires = Number.isFinite(reported) ? reported : Date.now() + sessionFallback;
  scheduledExpiryAt = expires;
  expiryTimer = window.setTimeout(() => {
    clearAuthState();
    authChannel?.postMessage({ type: "forced-unauthenticated", reason: "expired" });
  }, Math.max(0, expires - Date.now()));
}
function setAuthenticated(record: Learner, expiresAt?: string) {
  pb.authStore.save("", record);
  state = { status: "ready", record, error: null };
  scheduleExpiry(expiresAt);
  notify();
}
function parseMe(data: unknown): MeResult {
  if (!data || typeof data !== "object") return { kind: "retryable", error: new Error("Invalid auth response") };
  const response = data as { record?: unknown; session_expires_at?: unknown };
  const parsed = learnerSchema.safeParse(response.record);
  if (!parsed.success) return { kind: "retryable", error: new Error("Invalid learner response") };
  return {
    kind: "authenticated",
    record: parsed.data as Learner,
    expiresAt: typeof response.session_expires_at === "string" ? response.session_expires_at : undefined,
  };
}
export async function fetchAuthMe(): Promise<MeResult> {
  try {
    const response = await fetch(apiUrl("/api/auth/me"), { credentials: "include", headers: { "X-Requested-With": "fetch" } });
    if (response.status === 401) return { kind: "unauthenticated" };
    if (!response.ok) return { kind: "retryable", error: new Error(`Auth bootstrap failed: ${response.status}`) };
    return parseMe(await response.json());
  } catch (error) {
    return { kind: "retryable", error: error instanceof Error ? error : new Error("Network error") };
  }
}
export async function bootstrapAuth(): Promise<MeResult> {
  if (state.status === "ready" && state.record) return { kind: "authenticated", record: state.record };
  if (state.status === "ready" && !state.record && !state.error) return { kind: "unauthenticated" };
  if (bootstrapPromise) return bootstrapPromise;
  state = { ...state, status: "loading", error: null };
  notify();
  bootstrapPromise = fetchAuthMe().then((result) => {
    if (result.kind === "authenticated") setAuthenticated(result.record, result.expiresAt);
    else if (result.kind === "unauthenticated") clearAuthState(false);
    else { state = { ...state, status: "error", error: result.error }; notify(); }
    return result;
  }).finally(() => { bootstrapPromise = null; });
  return bootstrapPromise;
}
export type AuthErrorKind = "invalid-credentials" | "retryable";
export function classifyAuthError(error: unknown): AuthErrorKind {
  const status = typeof error === "object" && error !== null && "status" in error ? Number((error as { status?: unknown }).status) : 0;
  return status >= 400 && status < 500 ? "invalid-credentials" : "retryable";
}
export async function login(email: string, password: string): Promise<Learner> {
  try {
    const result = await pb.collection("learners").authWithPassword(email, password);
    const parsed = learnerSchema.safeParse(result.record);
    if (!parsed.success) throw new Error("Invalid learner response");
    const record = parsed.data as Learner;
    setAuthenticated(record);
    authChannel?.postMessage({ type: "login" });
    return record;
  } catch (error) {
    const kind = classifyAuthError(error);
    const classified = new Error(kind);
    Object.defineProperty(classified, "status", {
      value: kind === "invalid-credentials" ? 400 : 503,
      enumerable: false,
    });
    classified.cause = error;
    throw classified;
  }
}
export async function logout(): Promise<void> {
  try {
    await fetch(apiUrl("/api/auth/logout"), { method: "POST", credentials: "include", headers: { "X-Requested-With": "fetch" } });
  } catch {
    // The browser is signed out locally even when the server is unavailable.
  } finally {
    clearAuthState();
  }
}
