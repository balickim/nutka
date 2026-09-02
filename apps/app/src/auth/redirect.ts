const fallbackRedirect = "/";

export function getSafeRedirect(rawRedirect: string | null | undefined): string {
  if (!rawRedirect || !rawRedirect.startsWith("/") || rawRedirect.startsWith("//")) {
    return fallbackRedirect;
  }
  if (/^[\u0000-\u001f\\]/.test(rawRedirect) || rawRedirect.includes("\\")) {
    return fallbackRedirect;
  }
  try {
    const origin =
      typeof window === "undefined" ? "http://nutka.local" : window.location.origin;
    const parsed = new URL(rawRedirect, origin);
    if (parsed.origin !== origin || !parsed.pathname.startsWith("/")) {
      return fallbackRedirect;
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return fallbackRedirect;
  }
}

export function getRedirectFromLocation(search = window.location.search): string {
  return getSafeRedirect(new URLSearchParams(search).get("redirect"));
}
