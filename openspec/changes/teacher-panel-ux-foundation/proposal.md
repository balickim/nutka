## Why

The teacher panel renders every capability on one route. `/teachers` stacks the financial queue, a full commercial workspace per learner, and two lesson lists in a single scroll. Each learner card mounts four queries and three always-expanded administrative forms, so ten learners produce forty requests and thirty forms on load. The forms also expose the data model instead of the task: they ask for event identifiers, lesson identifiers, prices in minor units, and outcome lines in `date=outcome` syntax, and two destructive operations run through `window.prompt` and `window.confirm`. The panel does not scale past a few learners, and every new capability makes the single page worse. This change sets the structure before more capabilities arrive.

## What Changes

- **BREAKING**: `/teachers` stops rendering the combined dashboard. It becomes the "Today" screen: dated lessons plus a decision queue. Existing panel content moves to dedicated routes.
- Add `/teachers/students` (roster) and `/teachers/students/$id` (single learner). Commercial controls move there, so the app loads one learner's commercial data at a time.
- Add `/teachers/billing` as the home of the financial work queue, which today occupies the middle of the dashboard.
- Add `/teachers/calendar` for the week grid. `/teachers/availability` keeps its current purpose.
- Replace the learner card's three expanded forms with a state summary. The roster shows plan, weekly slot, and settlement state as read-only text.
- Split teacher write operations into three exposure tiers. Daily actions stay inline. Periodic actions move into a side-panel flow with a confirmation summary. Corrective actions move behind an advanced-operations disclosure and require a reason.
- Remove raw identifier and minor-unit inputs from the default path. Lesson conversion uses a selectable lesson list. Prices use a major-unit field. Event corrections select an event from history.
- Replace `window.prompt` and `window.confirm` with a dialog component that names the consequence.
- Define view states (loading, empty, error, ready) and mutation feedback as a shared contract, so each data block and each write reports the same way.
- Add design tokens for space, radius, type scale, and semantic color to `styles.css`, and reduce the hero heading so the first screen carries content.

## Capabilities

### New Capabilities
- `teacher-workspace-navigation`: The teacher route spaces, what each screen answers, which data each screen loads, and how the panel moves between them.
- `teacher-action-tiers`: The three exposure tiers for teacher write operations, the input rules that keep domain identifiers out of the default path, and the confirmation rules for destructive writes.
- `ui-view-states`: The required states for every data block, the feedback contract for every mutation, and the dialog contract for irreversible actions.

### Modified Capabilities
- `calendar-panels`: The teacher panel requirement currently binds the whole teacher workflow to `/teachers`. It changes to describe `/teachers` as the Today screen and to delegate availability, roster, learner detail, and billing to their own routes.

## Impact

- `apps/app/src/router.tsx`: four new routes, each with the existing persona guard.
- `apps/app/src/views/TeacherView.tsx`: splits into per-route views; `TeacherDashboard` is removed.
- `apps/app/src/views/teacher/TeacherCommercialWorkspace.tsx`: splits into a roster view and a learner detail view. `CommercialCard` no longer fetches per-learner commercial data from a list.
- `apps/app/src/views/teacher/TeacherFinancialWorkspace.tsx`: becomes the `/teachers/billing` view and feeds the Today decision queue.
- `apps/app/src/components/ScheduleBits.tsx`: gains the shared state and feedback components.
- `apps/app/src/styles.css`: gains design tokens.
- `apps/app/src/query/keys.ts`: learner-scoped keys replace the list-mounted package and contract queries defined inline today.
- `docs/constitutions/`: a new frontend view-state constitution records the shared contract.
- No backend, API contract, or authorization change. All routes keep the teacher persona guard.
