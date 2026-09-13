// Parses strict RFC3339 UTC instants and localizes them only when the UI renders concrete schedule values.

const utcInstantPattern = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(?:Z|[+-]00:00)$/;

export function parseUtcInstant(value: unknown): Date {
  const match = typeof value === "string" ? utcInstantPattern.exec(value) : null;
  if (!match) {
    throw new Error("Expected an RFC3339 UTC instant");
  }
  const parsed = new Date(value as string);
  const components = [
    parsed.getUTCFullYear(),
    parsed.getUTCMonth() + 1,
    parsed.getUTCDate(),
    parsed.getUTCHours(),
    parsed.getUTCMinutes(),
    parsed.getUTCSeconds(),
  ];
  const expected = match.slice(1, 7).map(Number);
  if (!Number.isFinite(parsed.getTime()) || components.some((component, index) => component !== expected[index])) {
    throw new Error("Expected an RFC3339 UTC instant");
  }
  return parsed;
}

export function formatUtcInstant(value: Date): string {
  if (!(value instanceof Date) || !Number.isFinite(value.getTime())) {
    throw new Error("Expected a valid instant");
  }
  return value.toISOString();
}

export function localizeUtcInstant(
  value: unknown,
  locale?: string,
  options?: Intl.DateTimeFormatOptions,
): string {
  return new Intl.DateTimeFormat(locale, options).format(parseUtcInstant(value));
}

export function isUtcInstant(value: unknown): value is string {
  try {
    parseUtcInstant(value);
    return true;
  } catch {
    return false;
  }
}
