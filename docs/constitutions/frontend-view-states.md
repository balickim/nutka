# Frontend view state constitution

This document defines how the Nutka React application presents data states, reports writes, and exposes teacher operations.
It does not change any backend contract in `docs/api/`.

## Data block states

- A block that reads server data expresses loading, empty, error, and ready.
- The loading state uses `Skeleton` and reserves the layout of the ready state.
- The empty state uses `EmptyState`, holds one sentence, and holds one control when an action can change the state.
- The error state uses `ApiFeedback`, which renders the Polish message for the stable error code.
- A block never renders a raw HTTP status, a transport message, or an untranslated server detail.
- A background refresh keeps the last successful data until the refresh settles.

## Mutation feedback

- A write reports both outcomes.
- A control that runs a write uses `ActionButton`, which keeps the label, keeps the size, and sets `aria-busy`.
- A successful write calls `useToast().notify` with one sentence that names what changed.
- A failed write renders `ApiFeedback` next to the control that failed.
- A failed write keeps the entered values so the teacher can correct them.

## Dialogs

- An action the panel cannot undo confirms through `ConfirmDialog`.
- The dialog names the consequence with concrete dates or amounts.
- The confirming control carries the action name and never reads as a generic acknowledgement.
- The dialog traps focus, closes on the escape key, and returns focus to the control that opened it.
- The application does not call `window.confirm` or `window.prompt`.

## Teacher action tiers

- Every teacher write belongs to one tier.
- Tier A covers daily work: lesson outcome, booking, settlement. A tier A control sits on the row it affects and completes in one step.
- Tier B covers periodic work: contract activation, package purchase, slot change, notice, renewal, price change. A tier B control opens `FlowPanel`, which states the effect before the write.
- Tier C covers corrective work: event correction, backdated activation, early contract end, package closure with refund. A tier C control sits inside `AdvancedOperations`, which stays collapsed and requires a reason.
- The reason of a tier C write travels to the API and appears in the history entry.

## Input rules

- Outside `AdvancedOperations`, no control asks for an identifier, a minor-unit amount, or a delimiter-encoded list.
- Lesson selection uses `LessonPicker`, which lists the assignment's eligible lessons by date.
- Event selection uses `EventPicker`, which lists the assignment's history entries.
- Money uses `apps/app/src/money.ts`. A field accepts a major-unit value and shows the currency.
- The application converts to minor units at the request boundary only.

## Policy presentation

- A control governed by a business policy prevents an invalid value through its own constraints.
- The rule text sits in `PolicyHint` next to the field it governs, not as a paragraph above the form.
- Policy values come from the authenticated policy read and never from a literal.

## Vocabulary

- The interface names a teacher-learner relationship as the learner.
- The interface uses one term for each concept on every screen.
- Polish copy lives in `apps/app/src/api/copy.ts`.

## Design tokens

- The file `apps/app/src/styles.css` defines every space, type, radius, and color token in `:root`.
- A rule references a token and holds no literal color, spacing, or radius value.
- The space scale uses four-pixel steps. The type scale holds seven steps and one display step.
- A status uses a color and a word together, so color alone never carries the meaning.
