// Computes practice day choices, the 4-week practice grid, and the calm summary lines from local dates in the teacher timezone.

import { formatLocalDate } from "../../../time/schedule";
import type { PracticeSummary } from "../../../api/practice";

export type DayChoice = { value: string; label: string };
export type GridDay = { date: string; practiced: boolean; future: boolean };

// Calendar arithmetic in UTC keeps whole days across daylight saving changes.
export function addDays(date: string, days: number): string {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(Date.UTC(year, month - 1, day + days)).toISOString().slice(0, 10);
}

export function dayChoices(today: string, backfillDays: number): DayChoice[] {
  return Array.from({ length: backfillDays + 1 }, (_, index) => {
    const value = addDays(today, -index);
    const label = index === 0 ? "Dziś" : index === 1 ? "Wczoraj" : formatLocalDate(value);
    return { value, label };
  });
}

// Four rows from Monday to Sunday. The last row holds today.
export function practiceGrid(today: string, practicedDays: string[]): GridDay[][] {
  const [year, month, day] = today.split("-").map(Number);
  const weekday = (new Date(Date.UTC(year, month - 1, day)).getUTCDay() + 6) % 7;
  const start = addDays(today, -weekday - 21);
  const practiced = new Set(practicedDays);
  return Array.from({ length: 4 }, (_, week) => Array.from({ length: 7 }, (_, offset) => {
    const date = addDays(start, week * 7 + offset);
    return { date, practiced: practiced.has(date), future: date > today };
  }));
}

function polishDays(count: number): string {
  return count === 1 ? "1 dzień" : `${count} dni`;
}

export function learnerPracticeLine(summary: PracticeSummary): string {
  if (summary.days === 0) return "Od ostatniej lekcji nie ma jeszcze zapisanych ćwiczeń.";
  const minutes = summary.minutes ? `, razem ${summary.minutes} min` : "";
  return `Od ostatniej lekcji: ${polishDays(summary.days)} z ćwiczeniem${minutes}.`;
}

export function teacherPracticeLine(summary: PracticeSummary): string {
  if (summary.days === 0) return "Od ostatniej lekcji: brak zapisanych ćwiczeń.";
  const minutes = summary.minutes ? `, ${summary.minutes} min` : "";
  return `Od ostatniej lekcji: ${polishDays(summary.days)} ćwiczeń${minutes}.`;
}
