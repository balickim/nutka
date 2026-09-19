// Derives Polish lesson-change previews from backend policy and assignment commercial balances without owning mutation rules.

import type { CommercialSummary, Lesson, PersonaRole, Policy } from "../api/contracts";

export type LessonChangeAction = "cancel" | "reschedule";

export function learnerChangeIsTimely(lesson: Lesson, policy: Policy, now = Date.now()): boolean {
  return Date.parse(lesson.start_at) - now >= policy.learner_change_cutoff_hours * 60 * 60 * 1000;
}

export function lessonChangeNotice(action: LessonChangeAction, lesson: Lesson, role: PersonaRole, policy: Policy, summary?: CommercialSummary, now = Date.now()): string {
  if (role === "teacher") return teacherNotice(action, lesson, policy);
  const timely = learnerChangeIsTimely(lesson, policy, now);
  const timing = timely ? `co najmniej ${policy.learner_change_cutoff_hours} godz. przed lekcją` : `mniej niż ${policy.learner_change_cutoff_hours} godz. przed lekcją`;
  if (action === "reschedule") return learnerRescheduleNotice(lesson, policy, timely, timing);
  return learnerCancellationNotice(lesson, summary, timely, timing);
}

function teacherNotice(action: LessonChangeAction, lesson: Lesson, policy: Policy): string {
  if (action === "reschedule") {
    if (lesson.plan_type === "package") return "Lekcja z pakietu przejdzie na nowy termin. Potwierdź zmianę terminu.";
    if (lesson.plan_type === "regular_contract") return "Zmiana nie wykorzysta limitu ucznia. Potwierdź zmianę terminu.";
    return "Stan rozliczenia pozostanie bez zmian. Potwierdź zmianę terminu.";
  }
  if (lesson.plan_type === "package") return `Lekcja wróci do pakietu, a jego ważność wzrośnie o ${policy.teacher_cancellation_extension_days} dni. Kontynuować?`;
  if (lesson.plan_type === "regular_contract") return "Lekcja nie będzie naliczona i nie wykorzysta limitu ucznia. Kontynuować?";
  return "Rozliczenie lekcji zostanie oznaczone jako nieobowiązujące. Kontynuować?";
}

function learnerRescheduleNotice(lesson: Lesson, policy: Policy, timely: boolean, timing: string): string {
  if (!timely) return `Zmiana jest zgłaszana ${timing}. Termin można tylko odwołać.`;
  if (lesson.plan_type === "package") return `Zmiana jest zgłaszana ${timing}. Lekcja z pakietu przejdzie na nowy termin. Potwierdź zmianę terminu.`;
  if (lesson.plan_type === "regular_contract") return `Zmiana jest zgłaszana ${timing}. Wykorzystasz jeden z ${policy.contract_monthly_reschedules} miesięcznych terminów zmiany. Nowy termin musi przypadać w ciągu ${policy.contract_replacement_deadline_days} dni. Potwierdź.`;
  return `Zmiana jest zgłaszana ${timing}. Rozliczenie pozostanie bez zmian. Potwierdź zmianę terminu.`;
}

function learnerCancellationNotice(lesson: Lesson, summary: CommercialSummary | undefined, timely: boolean, timing: string): string {
  if (lesson.plan_type === "ad_hoc") return `Odwołanie jest zgłaszane ${timing}. Zostanie zapisane bez kary i bez należności. Kontynuować?`;
  if (lesson.plan_type === "package") return timely
    ? `Odwołanie jest zgłaszane ${timing}. Lekcja wróci do pakietu. Kontynuować?`
    : `Odwołanie jest zgłaszane ${timing}. Lekcja z pakietu przepadnie. Kontynuować?`;
  const free = summary?.contract?.remaining_free_cancellations ?? 0;
  if (!timely) return `Odwołanie jest zgłaszane ${timing}. Lekcja pozostanie płatna. Kontynuować?`;
  if (free > 0) return `Odwołanie jest zgłaszane ${timing}. Wykorzystasz jedno z ${free} pozostałych bezpłatnych odwołań. Kontynuować?`;
  return `Odwołanie jest zgłaszane ${timing}. Limit został wykorzystany, więc lekcja pozostanie płatna. Kontynuować?`;
}
