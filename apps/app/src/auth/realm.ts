// Provides isolated in-memory browser session state for one Nutka persona.

import { z } from "zod";

import { apiUrl } from "../api/url";

const sessionFallback = 12 * 60 * 60 * 1000;
const intentHeaders = { "X-Requested-With": "fetch" };

export type RealmRecord = z.infer<typeof recordSchema> & Record<string, unknown>;
export type MeResult<T extends RealmRecord> =
  | { kind: "authenticated"; record: T; expiresAt?: string }
  | { kind: "unauthenticated" }
  | { kind: "retryable"; error: Error };
export type AuthState<T extends RealmRecord> = {
  status: "idle" | "loading" | "ready" | "error";
  record: T | null;
  error: Error | null;
};

const recordSchema = z.object({ id: z.string(), email: z.string() }).passthrough();

type RealmConfig<T extends RealmRecord> = {
  collection: string;
  mePath: string;
  logoutPath: string;
  channelName: string;
  displayName: (record: T) => string;
};

export type AuthRealm<T extends RealmRecord> = ReturnType<typeof createAuthRealm<T>>;

export function createAuthRealm<T extends RealmRecord>(config: RealmConfig<T>) {
  let state: AuthState<T> = { status: "idle", record: null, error: null };
  let bootstrapPromise: Promise<MeResult<T>> | null = null;
  let expiryTimer: ReturnType<typeof setTimeout> | undefined;
  let scheduledExpiryAt = 0;
  const listeners = new Set<() => void>();
  const channel = typeof BroadcastChannel === "undefined" ? null : new BroadcastChannel(config.channelName);

  function notify() {
    listeners.forEach((listener) => listener());
  }
  function clearExpiry() {
    if (expiryTimer !== undefined) clearTimeout(expiryTimer);
    expiryTimer = undefined;
    scheduledExpiryAt = 0;
  }
  function clearAuthState(broadcast = true) {
    clearExpiry();
    state = { status: "ready", record: null, error: null };
    notify();
    if (broadcast) channel?.postMessage({ type: "logout" });
  }
  function scheduleExpiry(expiresAt?: string) {
    clearExpiry();
    const reported = expiresAt ? Date.parse(expiresAt) : Number.NaN;
    const expires = Number.isFinite(reported) ? reported : Date.now() + sessionFallback;
    scheduledExpiryAt = expires;
    expiryTimer = setTimeout(() => {
      clearAuthState(false);
      channel?.postMessage({ type: "forced-unauthenticated" });
    }, Math.max(0, expires - Date.now()));
  }
  function setAuthenticated(record: T, expiresAt?: string) {
    state = { status: "ready", record, error: null };
    scheduleExpiry(expiresAt);
    notify();
  }
  function parseMe(data: unknown): MeResult<T> {
    if (!data || typeof data !== "object") return { kind: "retryable", error: new Error("Invalid auth response") };
    const response = data as { record?: unknown; session_expires_at?: unknown };
    const parsed = recordSchema.safeParse(response.record);
    if (!parsed.success) return { kind: "retryable", error: new Error("Invalid persona response") };
    return {
      kind: "authenticated",
      record: parsed.data as T,
      expiresAt: typeof response.session_expires_at === "string" ? response.session_expires_at : undefined,
    };
  }
  async function request(path: string, options?: RequestInit): Promise<Response> {
    return fetch(apiUrl(path), { ...options, credentials: "include", headers: { ...intentHeaders, ...options?.headers } });
  }
  async function fetchMe(): Promise<MeResult<T>> {
    try {
      const response = await request(config.mePath);
      if (response.status === 401) return { kind: "unauthenticated" };
      if (!response.ok) return { kind: "retryable", error: new Error(`Auth bootstrap failed: ${response.status}`) };
      return parseMe(await response.json());
    } catch (error) {
      return { kind: "retryable", error: error instanceof Error ? error : new Error("Network error") };
    }
  }
  async function bootstrap(): Promise<MeResult<T>> {
    if (state.status === "ready" && state.record) return { kind: "authenticated", record: state.record };
    if (state.status === "ready" && !state.record && !state.error) return { kind: "unauthenticated" };
    if (bootstrapPromise) return bootstrapPromise;
    state = { ...state, status: "loading", error: null };
    notify();
    bootstrapPromise = fetchMe().then((result) => {
      if (result.kind === "authenticated") setAuthenticated(result.record, result.expiresAt);
      else if (result.kind === "unauthenticated") clearAuthState(false);
      else {
        state = { ...state, status: "error", error: result.error };
        notify();
      }
      return result;
    }).finally(() => { bootstrapPromise = null; });
    return bootstrapPromise;
  }
  async function login(email: string, password: string): Promise<T> {
    try {
      const response = await request(`/api/collections/${config.collection}/auth-with-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ identity: email, password }),
      });
      if (!response.ok) throw Object.assign(new Error("Authentication failed"), { status: response.status });
      const parsed = parseMe(await response.json());
      if (parsed.kind !== "authenticated") throw new Error("Invalid persona response");
      setAuthenticated(parsed.record, parsed.expiresAt);
      channel?.postMessage({ type: "login" });
      return parsed.record;
    } catch (error) {
      const kind = classifyAuthError(error);
      const classified = new Error(kind);
      Object.defineProperty(classified, "status", { value: kind === "invalid-credentials" ? 400 : 503 });
      classified.cause = error;
      throw classified;
    }
  }
  async function logout() {
    try {
      await request(config.logoutPath, { method: "POST" });
    } catch {
      // Local state still clears when the server cannot be reached.
    } finally {
      clearAuthState();
    }
  }
  channel?.addEventListener("message", (event) => {
    const type = event.data?.type;
    if (type === "logout" || type === "forced-unauthenticated") clearAuthState(false);
    if (type === "login" && !state.record) {
      state = { status: "idle", record: null, error: null };
      void bootstrap();
    }
  });
  if (typeof document !== "undefined") {
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "visible" && scheduledExpiryAt > 0 && Date.now() >= scheduledExpiryAt) clearAuthState();
    });
  }
  return {
    subscribe(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener); }; },
    getState() { return state; },
    getDisplayName(record: T) { return config.displayName(record); },
    bootstrap,
    fetchMe,
    login,
    logout,
  };
}

export type AuthErrorKind = "invalid-credentials" | "retryable";
export function classifyAuthError(error: unknown): AuthErrorKind {
  const status = typeof error === "object" && error !== null && "status" in error ? Number((error as { status?: unknown }).status) : 0;
  return status >= 400 && status < 500 ? "invalid-credentials" : "retryable";
}
