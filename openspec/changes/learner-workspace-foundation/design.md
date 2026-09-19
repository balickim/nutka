## Context

See `proposal.md` for motivation and `specs/` for required behavior.

- `apps/app/src/views/home-view.tsx` owns the whole learner panel. It guards the session, loads calendar, policy, and slots, and renders one card per active assignment.
- Each card mounts commercial summary, history, and materials queries at once.
- `TeacherShell` already solves the same problem for teachers: one guard, one frame, one navigation, and a render prop with a shared context.
- The learner home fallback `/learners/calendar` is written in three places: `router.tsx`, `auth/redirect.ts`, and `views/login-view.tsx`.
- The backend resolves the persona caller in `materialsapi`, `ledgerapi`, `historyapi`, `commercialapi`, `schedulingapi`, `regularcontractapi`, and `businesspolicyapi`.

## Goals / Non-Goals

**Goals:**

- Give each later learner capability one route and one place in the navigation.
- Load a learner block's data only on the screen that shows it.
- Keep one implementation of the learner home route and of the persona caller rule.

**Non-Goals:**

- Change any backend endpoint, payload, or authorization rule.
- Add payments, notes, repertoire, practice, or messages screens. Later changes add them.
- Move the other six backend packages to `personaroute`. Each moves when a change touches it.
- Redesign the teacher panel.

## Decisions

### Mirror `TeacherShell` with `LearnerShell`

`views/learner/learner-shell.tsx` guards the learner session, loads calendar and policy, and passes `{ accountId, calendar, policy, assignments, assignment }` through a render prop. Each route view renders `<LearnerShell>` like teacher views render `<TeacherShell>`. TanStack Query dedupes the shared queries across route changes, so the shell does not need a layout route.

Alternative: a TanStack layout route with `<Outlet />`. Rejected because the teacher panel uses the render-prop shell, and two patterns for one rule add a second way to read the same context.

### Select the assignment through the `a` search parameter

A pure function `selectAssignment(assignments, requested)` returns the matching active assignment or the first active one. A search parameter survives reload and deep links, and it needs no client store. The switcher renders only for two or more active assignments, because most learners have one teacher.

### Keep slot queries on the lessons screen

`useLearnerSlots` moves from the shell to the lessons screen. Start does not need slots, so opening Start no longer requests them.

### One learner home constant

`auth/redirect.ts` exports `personaHome`. `router.tsx` and `login-view.tsx` import it. The learner value becomes `/learners`.

### Redirect the old route in `beforeLoad`

`/learners/calendar` keeps a route whose `beforeLoad` throws `redirect({ to: "/learners/lessons", replace: true })`. Saved bookmarks keep working and the browser history does not keep the old entry.

### Format date-only values without timezone shifts

`formatLocalDate("2026-10-14")` formats the wall-clock date with `Intl.DateTimeFormat("pl-PL", { weekday: "long", day: "numeric", month: "long", year: "numeric", timeZone: "UTC" })` on the UTC midnight of that date. The value has no instant, so a UTC formatter prevents a shift to the previous day. The existing `formatScheduleDate(\`${value}T12:00:00Z\`)` workaround in notice copy moves to this function.

### Readable plan summary

`CommercialSummaryPanel` moves to `views/learner/lessons/plan-summary.tsx`. A pure function `paymentFacts(summary)` returns only the non-zero sentences. It is testable without rendering.

### Learner tokens scoped by a class

The learner shell adds `learner-panel` to the panel root. `styles.css` sets `--text-md: 1.125rem` and a 44-pixel minimum height for buttons, links in navigation, and slot buttons inside `.learner-panel`. The teacher panel keeps its density.

### Navigation layout

`LearnerNav` lists the available screens: Start, Lekcje, Utwory. On viewports narrower than 720 pixels, the navigation is a fixed bottom bar with labels. On wider viewports it stays in the panel bar, like `TeacherNav`. Later changes append entries to one `screens` array.

### Extract `personaroute`

`internal/personaroute` exports:

- `type Role` with `Teacher` and `Learner`.
- `Caller(e, role) (*core.Record, error)`: the current `materialsapi.authenticatedCaller` rule.
- `RequireIntent(e) error`.
- `OwnedAssignment(app, id, role, accountID) (*core.Record, error)`: hides foreign assignments.
- `ErrUnauthenticated`, `ErrForbidden`, `ErrIntent`, and `WriteError(e, err) (bool, error)`, which writes the shared `401` and `403` bodies.

`materialsapi` keeps its own `errInvalid`, `errNotFound`, and error mapping for material codes, and delegates the three shared cases to `WriteError`.

Alternative: move all seven packages now. Rejected because it touches every payment and scheduling route without a behavior change and enlarges review.

### Constitution for learner content

`docs/constitutions/learner-content.md` states the rules shared by materials and later learner content: assignment scope, closed collections, persona routes, cascade delete, read-only inactive assignments, no business events, and UTC storage. `docs/api/materials.md` links to it.

## Risks / Trade-offs

- [Bookmarks to `/learners/calendar`] → The redirect keeps them working.
- [Playwright specs open `/learners/calendar`] → Update the specs to `/learners/lessons`, and keep one assertion for the redirect.
- [Bottom bar covers content on small screens] → Add bottom padding equal to the bar height to `.learner-panel main`.
- [Shared `personaroute` changes materials error bodies] → Keep codes and messages byte-identical. The existing `materials_api_test.go` protects them.

## Migration Plan

- The change is frontend routing plus a backend refactor with no data migration.
- Rollback reverts the commit. No stored state depends on it.
