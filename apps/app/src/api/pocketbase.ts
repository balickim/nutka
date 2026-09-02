import PocketBase, { BaseAuthStore } from "pocketbase";

const configuredApiUrl = import.meta.env.VITE_API_URL?.trim();
export const apiBaseUrl = configuredApiUrl || "/";
export const memoryAuthStore = new BaseAuthStore();
export const pb = new PocketBase(apiBaseUrl, memoryAuthStore);

pb.beforeSend = (_url, options) => {
  options.credentials = "include";
  options.headers = { ...options.headers, "X-Requested-With": "fetch" };
  return { options };
};

export function apiUrl(path: string): string {
  const base = apiBaseUrl.endsWith("/") ? apiBaseUrl.slice(0, -1) : apiBaseUrl;
  return `${base}${path}` || path;
}
