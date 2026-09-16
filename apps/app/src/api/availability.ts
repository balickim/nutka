// Provides teacher availability reads, previews, and resolved atomic commits through the shared transport.
import { apiRequest } from "./transport";
import type {
  AvailabilityCommitRequest,
  AvailabilityCommitResponse,
  AvailabilityException,
  AvailabilityPreview,
  AvailabilityProposal,
  AvailabilityRule,
} from "./contracts";
export const fetchAvailabilityRules = (signal?: AbortSignal) =>
  apiRequest<{ availability_rules: AvailabilityRule[] }>(
    "/api/teachers/availability/rules",
    { signal },
  );
export const fetchAvailabilityExceptions = (signal?: AbortSignal) =>
  apiRequest<{ availability_exceptions: AvailabilityException[] }>(
    "/api/teachers/availability/exceptions",
    { signal },
  );
export const previewAvailability = (
  body: AvailabilityProposal,
  signal?: AbortSignal,
) =>
  apiRequest<AvailabilityPreview>("/api/teachers/availability/preview", {
    method: "POST",
    body,
    signal,
  });
export const commitAvailability = (body: AvailabilityCommitRequest) =>
  apiRequest<AvailabilityCommitResponse>("/api/teachers/availability/commit", {
    method: "POST",
    body,
  });
