// Renders a failed read or write as its domain cause, with a retry control when the caller can retry.

import { getSchedulingErrorMessage } from "../api/scheduling";

export function ApiFeedback({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  if (!error) return null;
  return <div className="panel-error" role="alert"><span>{getSchedulingErrorMessage(error)}</span>{onRetry ? <button className="text-button" onClick={onRetry}>Spróbuj ponownie</button> : null}</div>;
}
