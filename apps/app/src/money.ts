// Converts between minor-unit amounts from the API and the major-unit values people read and type.

const MINOR_PER_MAJOR = 100;

// Input elements and JSON both use a dot, while Polish display uses a comma.
export function parseMajor(value: string): number | null {
  const normalized = value.trim().replace(",", ".");
  if (!/^-?\d+(\.\d{1,2})?$/.test(normalized)) return null;
  return Math.round(Number(normalized) * MINOR_PER_MAJOR);
}

export function toMajorInput(minor: number): string {
  return (minor / MINOR_PER_MAJOR).toFixed(2);
}

// Polish copy names the złoty with its everyday symbol instead of the ISO code.
const currencySymbols: Record<string, string> = { PLN: "zł" };

export function currencySymbol(currency: string): string {
  return currencySymbols[currency] ?? currency;
}

export function formatMoney(minor: number, currency: string): string {
  const negative = minor < 0;
  const absolute = Math.abs(minor);
  const units = Math.trunc(absolute / MINOR_PER_MAJOR);
  const fraction = String(absolute % MINOR_PER_MAJOR).padStart(2, "0");
  return `${negative ? "-" : ""}${units},${fraction} ${currencySymbol(currency)}`;
}
