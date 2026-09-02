import { apiUrl } from "./pocketbase";

export type HealthState = "checking" | "ready" | "unavailable";

export async function checkHealth(signal?: AbortSignal): Promise<HealthState> {
  try {
    const response = await fetch(apiUrl("/api/health"), { credentials: "include", signal });
    return response.ok ? "ready" : "unavailable";
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") throw error;
    return "unavailable";
  }
}
