// Owns persona session expiry timers and cross-tab event types without holding any account data.

import type { PersonaRole } from "../api/scheduling";

export type PersonaEvent = "login" | "logout" | "forced-unauthenticated";

const sessionFallback = 12 * 60 * 60 * 1000;
const channelNames: Record<PersonaRole, string> = { teacher: "nutka-teacher-auth", learner: "nutka-learner-auth" };

type ExpiryTimer = { handle: ReturnType<typeof setTimeout>; at: number; expire: () => void };

const timers = new Map<PersonaRole, ExpiryTimer>();
const listeners = new Set<(role: PersonaRole, event: PersonaEvent) => void>();
const channels = new Map<PersonaRole, BroadcastChannel>();

if (typeof BroadcastChannel !== "undefined") {
  (Object.keys(channelNames) as PersonaRole[]).forEach((role) => {
    const channel = new BroadcastChannel(channelNames[role]);
    channel.addEventListener("message", (message) => {
      const event = (message.data as { type?: PersonaEvent } | null)?.type;
      if (event) listeners.forEach((listener) => listener(role, event));
    });
    channels.set(role, channel);
  });
}

if (typeof document !== "undefined") {
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState !== "visible") return;
    Array.from(timers.entries()).forEach(([role, timer]) => {
      if (Date.now() < timer.at) return;
      clearExpiry(role);
      timer.expire();
    });
  });
}

export function publishPersonaEvent(role: PersonaRole, event: PersonaEvent): void {
  channels.get(role)?.postMessage({ type: event });
}

export function subscribePersonaEvents(listener: (role: PersonaRole, event: PersonaEvent) => void): void {
  listeners.add(listener);
}

export function scheduleExpiry(role: PersonaRole, expiresAt: string | undefined, expire: () => void): void {
  clearExpiry(role);
  const reported = expiresAt ? Date.parse(expiresAt) : Number.NaN;
  const at = Number.isFinite(reported) ? reported : Date.now() + sessionFallback;
  timers.set(role, { handle: setTimeout(expire, Math.max(0, at - Date.now())), at, expire });
}

export function clearExpiry(role: PersonaRole): void {
  const timer = timers.get(role);
  if (timer) clearTimeout(timer.handle);
  timers.delete(role);
}
