// Builds same-origin or configured API URLs without SDK state, authentication stores, or request side effects.

const configuredApiUrl = import.meta.env.VITE_API_URL?.trim();
export const apiBaseUrl = configuredApiUrl || "/";

export function apiUrl(path: string): string {
  const base = apiBaseUrl.endsWith("/") ? apiBaseUrl.slice(0, -1) : apiBaseUrl;
  return `${base}${path}` || path;
}
