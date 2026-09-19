// Converts schedule values at the UI boundary and keeps recurring wall-clock values in the teacher timezone.

import { formatUtcInstant, localizeUtcInstant, parseUtcInstant } from "./utc";

export const weekdayLabels = ["Niedziela", "Poniedziałek", "Wtorek", "Środa", "Czwartek", "Piątek", "Sobota"];

export function formatScheduleInstant(value: string, timezone?: string): string {
  return localizeUtcInstant(value, "pl-PL", { dateStyle: "medium", timeStyle: "short", ...(timezone ? { timeZone: timezone } : {}) });
}

export function formatScheduleDate(value: string, timezone?: string): string {
  return localizeUtcInstant(value, "pl-PL", { dateStyle: "medium", ...(timezone ? { timeZone: timezone } : {}) });
}

export function groupSlotsByLocalDate<T extends { start_at: string }>(items: T[], timezone?: string): Map<string, T[]> {
  const groups = new Map<string, T[]>();
  for (const item of items) {
    const date = new Intl.DateTimeFormat("en-CA", { ...(timezone ? { timeZone: timezone } : {}), year: "numeric", month: "2-digit", day: "2-digit" }).format(parseUtcInstant(item.start_at));
    const group = groups.get(date);
    if (group) group.push(item);
    else groups.set(date, [item]);
  }
  return groups;
}

export function utcToLocalInput(value: string, timezone?: string): string {
  const parts = new Intl.DateTimeFormat("en-CA", { ...(timezone ? { timeZone: timezone } : {}), year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23" }).formatToParts(parseUtcInstant(value));
  const fields = Object.fromEntries(parts.filter((part) => part.type !== "literal").map((part) => [part.type, part.value]));
  return `${fields.year}-${fields.month}-${fields.day}T${fields.hour}:${fields.minute}`;
}

function parseLocalInput(value: string): { localMs: number; parts: number[] } {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value);
  if (!match) throw new Error("Wybierz poprawną datę i godzinę.");
  const parts = match.slice(1).map(Number);
  const localMs = Date.UTC(parts[0], parts[1] - 1, parts[2], parts[3], parts[4]);
  const check = new Date(localMs);
  if (check.getUTCFullYear() !== parts[0] || check.getUTCMonth() !== parts[1] - 1 || check.getUTCDate() !== parts[2] || check.getUTCHours() !== parts[3] || check.getUTCMinutes() !== parts[4]) throw new Error("Wybierz poprawną datę i godzinę.");
  return { localMs, parts };
}

function timezoneOffsetMs(utcMs: number, timezone: string): number {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: timezone, year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23" }).formatToParts(new Date(utcMs));
  const fields = Object.fromEntries(parts.filter((part) => part.type !== "literal").map((part) => [part.type, part.value]));
  const localMs = Date.UTC(Number(fields.year), Number(fields.month) - 1, Number(fields.day), Number(fields.hour), Number(fields.minute), Number(fields.second));
  return localMs - Math.floor(utcMs / 1000) * 1000;
}

function localPartsAt(utcMs: number, timezone: string): number[] {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: timezone, year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23" }).formatToParts(new Date(utcMs));
  const fields = Object.fromEntries(parts.filter((part) => part.type !== "literal").map((part) => [part.type, part.value]));
  return [Number(fields.year), Number(fields.month), Number(fields.day), Number(fields.hour), Number(fields.minute)];
}

function resolveNamedTimezone(localMs: number, expected: number[], timezone: string): number {
  const offsets = new Set<number>();
  const initialGuess = localMs - timezoneOffsetMs(localMs, timezone);
  for (let delta = -3 * 60; delta <= 3 * 60; delta += 15) offsets.add(timezoneOffsetMs(initialGuess + delta * 60000, timezone));
  const candidates = [...offsets].map((offset) => localMs - offset).filter((utcMs) => localPartsAt(utcMs, timezone).every((part, index) => part === expected[index]));
  if (candidates.length === 0) throw new Error("Ta godzina nie istnieje w wybranej strefie.");
  if (candidates.length > 1) throw new Error("Ta godzina jest niejednoznaczna w wybranej strefie.");
  return candidates[0];
}

export function localInputToUtc(value: string, timezone?: string): string {
  const { localMs, parts } = parseLocalInput(value);
  if (!timezone) {
    const local = new Date(parts[0], parts[1] - 1, parts[2], parts[3], parts[4]);
    if (local.getFullYear() !== parts[0] || local.getMonth() !== parts[1] - 1 || local.getDate() !== parts[2] || local.getHours() !== parts[3] || local.getMinutes() !== parts[4]) throw new Error("Ta godzina nie istnieje w strefie lokalnej.");
    return formatUtcInstant(local);
  }
  return formatUtcInstant(new Date(resolveNamedTimezone(localMs, parts, timezone)));
}

export function futureExceptionDraft(now = new Date(), durationMinutes = 45): { start: string; end: string } {
  const startMs = Math.ceil((now.getTime() + 86400000) / 900000) * 900000;
  return { start: utcToLocalInput(formatUtcInstant(new Date(startMs))), end: utcToLocalInput(formatUtcInstant(new Date(startMs + durationMinutes * 60000))) };
}

export function formatWeeklySlot(value: string, timezone?: string): string {
  return localizeUtcInstant(value, "pl-PL", { weekday: "long", hour: "2-digit", minute: "2-digit", hourCycle: "h23", ...(timezone ? { timeZone: timezone } : {}) });
}

export function isSameLocalDay(value: string, reference: Date, timezone?: string): boolean {
  const format = (date: Date) => new Intl.DateTimeFormat("en-CA", { ...(timezone ? { timeZone: timezone } : {}), year: "numeric", month: "2-digit", day: "2-digit" }).format(date);
  return format(parseUtcInstant(value)) === format(reference);
}
