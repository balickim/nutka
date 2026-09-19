// Provides the only frontend Fetch API boundary: URL building, cookie credentials, mutation intent, JSON or multipart bodies, and typed errors.

import { apiUrl } from "./url";

export type ApiError = { code: string; message: string };
export type ApiRequestOptions = { method?: string; body?: unknown; signal?: AbortSignal };

const intentHeader = { "X-Requested-With": "fetch" };
const jsonHeader = { "Content-Type": "application/json" };
const genericError: ApiError = { code: "request_failed", message: "Coś poszło nie tak. Spróbuj ponownie." };

export class ApiRequestError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(error: ApiError, status: number) {
    super(error.message);
    this.name = "ApiRequestError";
    this.code = error.code;
    this.status = status;
  }
}

export function isRetryableFailure(error: unknown): boolean {
  if (!(error instanceof ApiRequestError)) return true;
  return error.status < 400 || error.status >= 500;
}

function requestInit({ method = "GET", body, signal }: ApiRequestOptions): RequestInit {
  const mutation = method !== "GET";
  // The browser sets the multipart boundary header for FormData bodies.
  const multipart = body instanceof FormData;
  return {
    method,
    credentials: "include",
    signal,
    headers: { ...(mutation ? intentHeader : {}), ...(body === undefined || multipart ? {} : jsonHeader) },
    ...(body === undefined ? {} : { body: multipart ? body : JSON.stringify(body) }),
  };
}

async function readError(response: Response): Promise<ApiError> {
  try {
    const body = (await response.json()) as Partial<ApiError>;
    if (typeof body.code === "string" && typeof body.message === "string") return body as ApiError;
  } catch {
    // Non-JSON failures keep the stable generic message.
  }
  return genericError;
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const response = await fetch(apiUrl(path), requestInit(options));
  if (!response.ok) throw new ApiRequestError(await readError(response), response.status);
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}
