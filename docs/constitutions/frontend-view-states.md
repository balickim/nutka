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
- The availability impact review opens in `ConfirmDialog`. The teacher resolves each conflict inside the dialog before the save.
- A learner slot choice opens `ConfirmDialog` with the date, duration, and teacher. The booking happens only after the learner confirms.
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

## Learner navigation

- Every learner screen renders inside `LearnerShell`, which requires the learner session.
- `LearnerShell` loads the learner session, business policy, and learner calendar once for every learner screen.
- The learner home is `/learners`. The Start screen shows the next lesson and links to other screens.
- `/learners/lessons` holds lessons, booking, plan summary, notice, history, and package lesson details.
- `/learners/pieces` holds the teacher materials. No other learner screen reads the materials list.
- `/learners/calendar` redirects to `/learners/lessons` and replaces the history entry.
- The `a` search parameter selects the assignment. A missing or unknown value selects the first active assignment.
- The assignment switcher appears only when the learner has two or more active assignments.
- Each learner screen shows data for the selected assignment only.
- A learner screen shows a date-only value as a long Polish date and never as `YYYY-MM-DD`.
- The plan summary shows a payment line only for a value that is not zero.
- `/learners/payments` shows the payment-due read of the selected assignment. No learner screen blocks an action because of an open payment.
- The Start screen shows the payment card only when the total due is above zero.
- The payments screen shows a ZBP transfer QR code only when transfer details exist and the total fits six digits in grosze.
- The transfer title names the learner and the open months or lesson dates and holds at most 32 characters.
- `/teachers/settings` holds the teacher transfer details form.
- The Start screen shows the latest lesson note of the selected assignment when one exists.
- `/learners/lessons` lists every lesson note of the selected assignment under the anchor `notatki`.
- A teacher Today card offers the note editor after the teacher records the outcome `completed`.
- The teacher learner view has a Notatki tab with the started, not cancelled lessons of the learner, newest first.
- The `.learner-panel` class raises the type scale and sets a 44-pixel minimum touch target.
- Below a 720-pixel viewport, the learner navigation is a fixed bar at the bottom of the screen.

## Input rules

- Outside `AdvancedOperations`, no control asks for an identifier, a minor-unit amount, or a delimiter-encoded list.
- Lesson selection uses `LessonPicker`, which lists the assignment's eligible lessons by date.
- Event selection uses `EventPicker`, which lists the assignment's history entries.
- Money uses `apps/app/src/money.ts`. A field accepts a major-unit value and shows the currency.
- The interface names the Polish złoty `zł` and never shows the ISO code `PLN`.
- The application converts to minor units at the request boundary only.

## Policy presentation

- A control governed by a business policy prevents an invalid value through its own constraints.
- The rule text sits in `PolicyHint` next to the field it governs, not as a paragraph above the form.
- Policy values come from the authenticated policy read and never from a literal.

## Vocabulary

- The interface names a teacher-learner relationship as the learner.
- The interface uses one term for each concept on every screen.
- The interface uses everyday words. It does not show technical terms such as "atomowo", "token", "horyzont", or "ad hoc".
- Polish copy lives in `apps/app/src/api/copy.ts`.
- A section help tooltip restates rules from the relevant constitution in everyday words. It does not add rules.

## Design tokens

- The package `packages/ui` (`@nutka/ui`) defines the shared palette, fonts, card radii, and shadow in `tokens.css`.
- The app and the landing page import `@nutka/ui` tokens, fonts, base typography, and component classes. Neither app copies a palette value.
- The shared component classes are `.brand`, `.eyebrow`, `.card`, `.btn`, `.btn-primary`, `.btn-accent`, `.btn-ghost`, `.btn-sm`, and `.marker`.
- The app uses `.btn-primary` for the main action, `.btn-ghost` for secondary actions, and `.text-button` for low-emphasis actions.
- The file `apps/app/src/styles.css` maps the shared palette to semantic app tokens in `:root` and defines the space and type scales.
- The landing page maps the shared tokens to Tailwind utilities with `@theme inline reference` in `apps/landing/src/styles/tokens.css`.
- The green `--color-grass` token appears only as a fill or under dark text. It never colors text.
- A rule references a token and holds no literal color, spacing, or radius value.
- The space scale uses four-pixel steps. The type scale holds seven steps and one display step.
- A status uses a color and a word together, so color alone never carries the meaning.
