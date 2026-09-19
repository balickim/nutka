## Why

The learner panel renders every capability on one route, `/learners/calendar`. Lessons, booking, plan balances, notice, materials, history, and token details stack in one scroll per teacher card. Raw ISO dates and zero-value payment counters reach the screen. Nutka targets adults and seniors, and the next planned learner capabilities (payments, lesson notes, repertoire, practice, messages) need their own screens. This change sets the learner structure before those capabilities arrive, as `teacher-panel-ux-foundation` did for teachers.

The backend also resolves persona sessions in seven packages with separate copies of the same rule. Five planned learner-content APIs would add five more copies.

## What Changes

- **BREAKING**: `/learners/calendar` stops rendering the learner panel. It redirects to `/learners/lessons`.
- Add a learner shell that guards every learner route, loads session, policy, and calendar once, and renders learner navigation.
- Add `/learners` (Start): the next lesson and short summaries that link to other screens.
- Add `/learners/lessons`: the current lesson list, booking, plan summary, notice, history, and token details.
- Add `/learners/pieces`: teacher materials, which later hold the repertoire.
- Add an assignment switcher that appears only when the learner has more than one active assignment.
- Change the learner home route after login from `/learners/calendar` to `/learners`.
- Format plan dates (`valid_through`, `start_on`, `end_on`) as localized long dates instead of raw ISO strings.
- Show only non-zero payment facts in the plan summary, with plain Polish sentences.
- Apply senior-friendly learner tokens: larger base text and 44-pixel touch targets.
- Add `internal/personaroute` in the backend as the one implementation of persona caller resolution, intent checks, and assignment ownership. Move `materialsapi` to it.
- Add the `docs/constitutions/learner-content.md` constitution for assignment-scoped learner content.

## Capabilities

### New Capabilities

- `learner-workspace-navigation`: The learner route spaces, what each screen answers, which data each screen loads, the assignment switcher, and readable date and payment copy.

### Modified Capabilities

- `calendar-panels`: The learner panel requirement binds the learner workflow to `/learners/calendar`. It changes to place lessons and booking on `/learners/lessons` and to redirect the old route.

## Impact

- `apps/app/src/router.tsx`: three new learner routes, one redirect, new learner home.
- `apps/app/src/views/home-view.tsx`: removed. Content moves to `apps/app/src/views/learner/*`.
- `apps/app/src/time/schedule.ts`: gains `formatLocalDate` for date-only wall-clock values.
- `apps/app/src/styles.css`: gains learner panel tokens.
- `apps/backend/internal/personaroute`: new package. `apps/backend/internal/materialsapi/auth.go` is removed.
- `docs/constitutions/learner-content.md`, `docs/constitutions/frontend-view-states.md`, `docs/auth.md`, `docs/README.md`.
- No API contract, payload, cookie, or authorization change.
