## 1. Design tokens and type scale

- [x] 1.1 Add `:root` tokens to `apps/app/src/styles.css`: space scale on 4px steps, seven-step type scale, three radii, and semantic colors (`--surface`, `--surface-raised`, `--border`, `--text`, `--text-muted`, `--accent`, `--danger`, `--success`, `--warning`). Preserve the current palette values.
- [x] 1.2 Rewrite every existing rule in `styles.css` to reference tokens, so no literal spacing, radius, or color value remains.
- [x] 1.3 Reduce the panel hero: drop the heading to the token scale's third step, merge the hero into the header bar, and remove the duplicated `eyebrow` in `PanelFrame`.
- [x] 1.4 Add a `:focus-visible` ring on the accent token for every interactive element.
- [x] 1.5 Add status badge styles for settled, pending, overdue, cancelled, and no-show, each pairing a color with a word.
- [x] 1.6 Verify `/teachers` and `/learners/calendar` render unchanged except for the hero.

## 2. Shared view-state and feedback components

- [x] 2.1 Create `apps/app/src/money.ts` with major-unit parsing, minor-unit conversion, and display formatting, plus tests for rounding and the zero case.
- [x] 2.2 Create `apps/app/src/components/Skeleton.tsx` that reserves the layout of its ready state.
- [x] 2.3 Extend `EmptyState` with an optional action slot and move it to `apps/app/src/components/EmptyState.tsx`.
- [x] 2.4 Create `apps/app/src/components/Toast.tsx` with a provider mounted at the router root and a `useToast` hook. No server state.
- [x] 2.5 Create `apps/app/src/components/ConfirmDialog.tsx` with focus trap, escape dismissal, focus return, a consequence line, and an action-named confirm control.
- [x] 2.6 Create `apps/app/src/components/PolicyHint.tsx` that discloses a policy rule next to the field it governs.
- [x] 2.7 Add a busy-state rule to shared buttons: keep the label and size, set `aria-busy`, show a spinner beside the label.
- [x] 2.8 Reduce `ScheduleBits.tsx` to schedule-specific pieces and update its importers.
- [x] 2.9 Add a test helper that renders a component with the QueryClient and Toast providers together.
- [x] 2.10 Confirm every touched file stays inside the 250 NLOC and CCN 10 budgets.

## 3. Query keys for learner-scoped commercial data

- [x] 3.1 Add `assignmentPackages` and `assignmentContracts` keys to `apps/app/src/query/keys.ts`.
- [x] 3.2 Remove the inline `packageListQuery` and `contractListQuery` builders from `TeacherCommercialWorkspace.tsx` and use the new keys.
- [x] 3.3 Update the key table in `docs/constitutions/frontend-query-state.md` with both keys.
- [x] 3.4 Extend `apps/app/src/query/keys.test.ts` to cover the new keys.

## 4. Billing route

- [x] 4.1 Add the `/teachers/billing` route in `router.tsx` with the existing teacher persona guard.
- [x] 4.2 Move `TeacherFinancialWorkspace` to the billing route and give each queue entry a learner name, an amount from `money.ts`, and a cause.
- [x] 4.3 Apply the four view states to each queue block, with an empty state that names the next destination.
- [x] 4.4 Report every settlement write through `Toast` and show failures next to the failing control.

## 5. Roster route

- [x] 5.1 Add the `/teachers/students` route with the teacher persona guard.
- [x] 5.2 Write one derivation function that maps `CalendarResponse.assignments` and `commercial_summaries` to roster rows (learner name, active plan, weekly slot, settlement state, active flag), with a test.
- [x] 5.3 Render the roster from that function only. Issue no per-assignment request from this screen.
- [x] 5.4 Link each row to `/teachers/students/$id` keyed by assignment identifier.
- [x] 5.5 Rename the user-facing term from assignment to learner across roster copy in `copy.ts`.

## 6. Learner detail route and action tiers

- [x] 6.1 Add the `/teachers/students/$id` route with the teacher persona guard, and show an authorization error for an assignment the teacher does not own.
- [x] 6.2 Split the learner detail into tabs: overview, plan, history. One file per tab.
- [x] 6.3 Create `apps/app/src/components/FlowPanel.tsx` for tier B flows: collect fields, then show a plain-language summary before the write.
- [x] 6.4 Create `apps/app/src/components/AdvancedOperations.tsx`: collapsed by default, requires a non-empty reason, labels results as corrections.
- [x] 6.5 Move contract activation into a tier B flow whose summary states the first lesson date, recurring slot, price per lesson, and end date.
- [x] 6.6 Move package purchase and renewal into a tier B flow.
- [x] 6.7 Move recurring-slot change, notice, and future price amendment into tier B flows. Price uses `money.ts` in major units with the currency shown.
- [x] 6.8 Move event correction, backdated activation, early contract end, and package closure with refund into `AdvancedOperations`.
- [x] 6.9 Replace the ad hoc lesson identifier fields with a selectable list of the assignment's eligible ad hoc lessons showing their dates.
- [x] 6.10 Replace the event identifier fields with a selection from the assignment's history entries.
- [x] 6.11 Replace the `date=outcome` textarea with a per-date outcome control in the backdated-activation flow.
- [x] 6.12 Remove every `window.prompt` and `window.confirm` from the teacher views and use `ConfirmDialog`.
- [x] 6.13 Replace policy paragraphs above forms with `PolicyHint` beside the governed field, and enforce each limit through the control's own constraints.
- [x] 6.14 Verify the detail route loads commercial data for one assignment only.

## 7. Today screen

- [x] 7.1 Replace `TeacherDashboard` with the Today screen at `/teachers`: the current local day's lessons in start order, plus the decision queue.
- [x] 7.2 Derive the decision queue from the same unresolved-work source as billing, and link each item to `/teachers/billing`.
- [x] 7.3 Give each today lesson its tier A outcome controls, reported through `Toast`.
- [x] 7.4 Add empty states to both blocks that name the next useful destination.
- [x] 7.5 Delete `TeacherDashboard` and any section it alone rendered.

## 8. Calendar route and navigation

- [x] 8.1 Add the `/teachers/calendar` route and move the near-term and later contract lesson lists to it.
- [x] 8.2 Replace the two-link `panel-nav` with navigation covering all six screens, marking the current one.
- [x] 8.3 Make the navigation reachable without scrolling below 768 pixels.
- [x] 8.4 Verify the Today screen and its outcome controls are usable at 375 pixels wide.

## 9. Learner panel alignment

- [x] 9.1 Replace the learner notice `window.confirm` in `HomeView.tsx` with `ConfirmDialog` stating the resolved end date.
- [x] 9.2 Apply the four view states and `Toast` feedback to the learner booking and notice flows.
- [x] 9.3 Format every learner-facing amount through `money.ts`.

## 10. Documentation and verification

- [x] 10.1 Add a frontend view-state constitution under `docs/constitutions/` covering the four data-block states, the mutation feedback contract, the dialog contract, and the one-term-per-concept rule. Link it from `docs/README.md`.
- [x] 10.2 Record the tier placement rule in the same constitution: tier A inline, tier B in a flow, tier C behind advanced operations with a reason.
- [x] 10.3 Run `./tools/code_quality/check.py check` and resolve every finding.
- [x] 10.4 Run the frontend test suite and the build, and confirm both pass.
- [x] 10.5 Run `openspec validate teacher-panel-ux-foundation --strict`.
