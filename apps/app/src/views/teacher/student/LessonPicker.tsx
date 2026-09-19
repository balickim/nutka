// Selects existing ad hoc lessons by their dates, so no flow asks the teacher to type a lesson identifier.

import type { Lesson } from "../../../api/contracts";
import { EmptyState } from "../../../components/EmptyState";
import { formatScheduleInstant } from "../../../time/schedule";

export function LessonPicker({ lessons, selected, onChange }: { lessons: Lesson[]; selected: string[]; onChange: (value: string[]) => void }) {
  if (lessons.length === 0) return <EmptyState>Brak pojedynczych lekcji do zamiany.</EmptyState>;
  const toggle = (id: string) => onChange(selected.includes(id) ? selected.filter((item) => item !== id) : [...selected, id]);
  return <fieldset className="picker">
    <legend>Pojedyncze lekcje do zamiany</legend>
    {lessons.map((lesson) => <label className="check-label" key={lesson.id}>
      <input type="checkbox" checked={selected.includes(lesson.id)} onChange={() => toggle(lesson.id)} />
      {formatScheduleInstant(lesson.start_at)}
    </label>)}
  </fieldset>;
}
