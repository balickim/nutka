// Validates same-origin redirects and restricts post-login navigation to one persona route tree.

const fallbackRedirect = "/";

export type RedirectPersona = "learner" | "teacher";

export const personaHome: Record<RedirectPersona, string> = { teacher: "/teachers", learner: "/learners" };

export function getSafeRedirect(rawRedirect: string | null | undefined, fallback = fallbackRedirect): string {
  if (!rawRedirect || !rawRedirect.startsWith("/") || rawRedirect.startsWith("//")) {
    return fallback;
  }
  if (/^[\u0000-\u001f\\]/.test(rawRedirect) || rawRedirect.includes("\\")) {
    return fallback;
  }
  try {
    const origin =
      typeof window === "undefined" ? "http://nutka.local" : window.location.origin;
    const parsed = new URL(rawRedirect, origin);
    if (parsed.origin !== origin || !parsed.pathname.startsWith("/")) {
      return fallback;
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return fallback;
  }
}

export function getPersonaRedirect(
  rawRedirect: string | null | undefined,
  persona: RedirectPersona,
): string {
  const root = personaHome[persona];
  const fallback = personaHome[persona];
  const candidate = getSafeRedirect(rawRedirect, fallback);
  try {
    const pathname = new URL(candidate, "http://nutka.local").pathname;
    return pathname === root || pathname.startsWith(`${root}/`) ? candidate : fallback;
  } catch {
    return fallback;
  }
}
