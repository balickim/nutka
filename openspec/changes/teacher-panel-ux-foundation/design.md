## Context

See `proposal.md` — Why. The constraints that shape the approach:

- `docs/constitutions/frontend-query-state.md` already fixes query keys, cache rules, the transport boundary, and the presentation rules. This change adds screens and components; it does not move the cache authority. New per-screen keys go into `apps/app/src/query/keys.ts` like every other key.
- `AGENTS.md` sets a 250 NLOC budget per file and CCN 10 per function. `TeacherCommercialWorkspace.tsx` is 176 lines of very dense JSX and already carries seven distinct control groups. Splitting it by route is the structural fix the budget implies.
- Policy values come from the authenticated policy read. Every limit shown or enforced in a control must read from `Policy`, never from a literal.
- The backend exposes no new endpoint for this change. Today and Billing both read the existing unresolved-work and financial-work endpoints. The roster reads the existing calendar response, which already carries `assignments` and `commercial_summaries`.
- Routes use TanStack Router with a `beforeLoad` persona guard. The guard pattern in `router.tsx` is reused unchanged.

## Goals / Non-Goals

**Goals:**

- One screen per question, with each screen's data cost proportional to what it shows.
- A learner's commercial data is fetched when that learner is open, not when the roster renders.
- A shared vocabulary of view states and feedback that later features inherit for free.
- Design tokens that make future spacing and color decisions mechanical.

**Non-Goals:**

- No backend, API contract, or authorization change.
- No change to the learner panel beyond adopting the shared view-state and dialog components.
- No component library adoption. The app has no UI dependency today and this change does not add one.
- No calendar grid rendering engine. `/teachers/calendar` starts as the existing lesson lists behind their own route; the week grid is a later change.
- No offline or optimistic behavior for compound commercial writes. The query constitution forbids it.

## Decisions

### Route split over in-page tabs

Each screen is a route with its own guard and its own queries. Alternative: keep `/teachers` and add client-side tabs. Rejected because tabs keep all data mounted, so the N-query problem survives; a route also gives a shareable URL for one learner, which the teacher needs to return to a specific case.

`/teachers/students/$id` uses the assignment identifier as the path parameter, not the learner account identifier. The assignment is what every commercial endpoint already addresses, and one learner can hold more than one assignment.

### Roster reads the calendar response; detail reads commercial endpoints

The roster derives plan, weekly slot, and settlement state from `CalendarResponse.assignments` and `CalendarResponse.commercial_summaries`, which one existing request already returns. It issues no per-assignment request. The detail route keeps the current four queries but for one assignment.

This removes the inline `packageListQuery` and `contractListQuery` builders from `TeacherCommercialWorkspace.tsx`, which violate the constitution's rule that only `keys.ts` builds key arrays. They move to `keys.ts` as `assignmentPackages` and `assignmentContracts`.

Alternative: a new roster summary endpoint. Rejected as premature — the calendar response already carries the fields, and adding an endpoint widens the change beyond the frontend.

### Tiers are a placement rule, not a new abstraction

Tier A, B, and C do not become a framework, a registry, or a config object. They are a rule about where a control lives: inline, in a `<FlowPanel>`, or inside `<AdvancedOperations>`. Alternative: a declarative action registry that renders each operation from metadata. Rejected under YAGNI — it would add indirection over roughly twenty operations that each need bespoke fields anyway.

The reason requirement for tier C is enforced at the control, by disabling the confirm until a reason is present, which matches how `closePackage` and `endContractEarly` already gate on `reason.trim()`. The difference is that the reason field now lives in the dialog rather than floating in the panel.

### Shared state components extend `ScheduleBits`, then split

`EmptyState` and `ApiFeedback` already live in `apps/app/src/components/ScheduleBits.tsx`. The new `Skeleton`, `Toast`, `ConfirmDialog`, `FlowPanel`, and `AdvancedOperations` push that file past the NLOC budget, so components move to `apps/app/src/components/` as one file per component and `ScheduleBits.tsx` keeps only the schedule-specific pieces.

`Toast` needs one provider at the router root and a `useToast` hook. It holds no server state, so it stays outside the query cache.

### Tokens replace literals in one pass

`styles.css` is 114 lines. Tokens are added at `:root` and existing rules are rewritten to reference them in the same commit, so the file never holds two spacing systems. Scale: space on 4px steps, a type scale of seven steps, three radii, and semantic colors (`--surface`, `--surface-raised`, `--border`, `--text`, `--text-muted`, `--accent`, `--danger`, `--success`, `--warning`). The existing palette values are preserved; only their names change.

Status badges need colour plus a word, because colour alone fails for colour-blind users and in the printed view a teacher may use.

### Money and identifiers convert at the boundary

A `money.ts` module in `apps/app/src/` converts between a major-unit string and minor units, and formats for display. Every price field uses it. This removes the `(value / 100).toFixed(2)` repetition that currently appears in at least four components with two different precisions.

### Policy help replaces policy paragraphs

Each policy sentence currently rendered as `supporting-copy` above a form becomes the content of a `<PolicyHint>` next to the field it governs, and the control's own constraints (`min`, `max`, `step`, option list) enforce the rule. The copy itself stays in `copy.ts`.

## Risks / Trade-offs

- **The route split touches every teacher view at once, so a partial merge leaves the panel broken.** → Sequence the tasks so routes and views land together per screen: add the route, move its content, delete the old section. `/teachers` keeps rendering the old dashboard until the last screen moves.
- **Deriving roster state from the calendar response couples the roster to that payload's shape.** → The derivation lives in one function with a test. If the payload stops carrying `commercial_summaries`, one function changes.
- **`/teachers` changing meaning breaks a teacher's bookmark of the old dashboard.** → Acceptable: the app has no public users and no external links. The title contract in `calendar-panels` is unchanged, so document titles still resolve.
- **Tokens and view states applied across both personas risk visual regressions in the learner panel.** → Tokens preserve current values, so the learner panel should render identically; verify `/learners/calendar` after the token pass.
- **Splitting components into one file each raises the file count noticeably.** → Preferred over the alternative, which is a single file that exceeds the NLOC budget and mixes unrelated reasons to change.
- **A `Toast` provider at the root adds a global that tests must mount.** → Provide a test helper that renders with the provider, in the same place the QueryClient test wrapper lives.

## Migration Plan

The change ships in screen-sized increments, each one independently shippable:

1. Tokens and the type scale. No behavior change, both personas verified.
2. Shared components (`Skeleton`, `Toast`, `ConfirmDialog`, `PolicyHint`, `EmptyState` with an action slot), plus `money.ts`. Not yet consumed everywhere.
3. `/teachers/billing` and `/teachers/students` routes. The old dashboard keeps its copies until step 5.
4. `/teachers/students/$id`, with the commercial workspace moved and split by tier.
5. `/teachers` becomes Today; the old `TeacherDashboard` is deleted and `calendar-panels` is updated.
6. `/teachers/calendar` receives the lesson lists; navigation covers all six screens.
7. Constitution and spec updates.

Rollback for any step is the revert of that step, because no step changes stored data or an API contract.

## Open Questions

- Whether the Today decision queue should cap its length or show every unresolved item. The unresolved-work payload is unbounded in principle but small in practice; decide when real data exists.
- Whether `/teachers/calendar` eventually replaces `/teachers/availability` with a single editable grid. Out of scope here; the spec keeps them separate.
