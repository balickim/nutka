## 1. Backend persona route rule

- [x] 1.1 Add `apps/backend/internal/personaroute` with `Role`, `Caller`, `RequireIntent`, `OwnedAssignment`, shared errors, and `WriteError`
- [x] 1.2 Add `personaroute` tests for opposite-realm cookie, unverified account, missing intent, and foreign assignment
- [x] 1.3 Move `materialsapi` to `personaroute`, remove `materialsapi/auth.go`, and keep material error codes and messages unchanged
- [x] 1.4 Run `go build ./...` and `go test ./...` in `apps/backend`

## 2. Learner home and routing

- [x] 2.1 Export one `personaHome` from `auth/redirect.ts` with learner value `/learners`, and use it in `router.tsx`, `getPersonaRedirect`, and `login-view.tsx`
- [x] 2.2 Update `redirect.test.ts` for the new learner fallback
- [x] 2.3 Add routes `/learners`, `/learners/lessons`, and `/learners/pieces` with the learner guard
- [x] 2.4 Change `/learners/calendar` to a `beforeLoad` redirect to `/learners/lessons` with `replace`

## 3. Learner shell and assignment selection

- [x] 3.1 Add pure `selectAssignment(assignments, requested)` with tests for missing, foreign, and inactive values
- [x] 3.2 Add `views/learner/learner-shell.tsx` with session guard, calendar and policy queries, `PanelFrame`, and render-prop context
- [x] 3.3 Add `LearnerNav` with Start, Lekcje, and Utwory, and an assignment switcher shown only for two or more active assignments
- [x] 3.4 Render the no-active-assignment state in the shell

## 4. Learner screens

- [x] 4.1 Add `views/learner/start-view.tsx` with the next scheduled lesson and links to lessons and pieces
- [x] 4.2 Move lesson list, booking, slot picker, plan summary, notice, history, and token details to `views/learner/lessons-view.tsx` and `views/learner/lessons/*`
- [x] 4.3 Move the slot queries into the lessons screen for the selected assignment only
- [x] 4.4 Add `views/learner/pieces-view.tsx` with the materials block
- [x] 4.5 Remove `views/home-view.tsx`

## 5. Readable copy and tokens

- [x] 5.1 Add `formatLocalDate` to `time/schedule.ts` with tests for a date near a DST change and for a Warsaw day boundary
- [x] 5.2 Use `formatLocalDate` for `valid_through`, `start_on`, `end_on`, and the notice consequence
- [x] 5.3 Add pure `paymentFacts(summary)` with a test for all-zero and mixed values, and use it in the plan summary
- [x] 5.4 Add `.learner-panel` tokens and the narrow-viewport bottom navigation to `styles.css`

## 6. Browser tests

- [x] 6.1 Update `tests/auth.spec.ts` and `tests/scheduling.spec.ts` to open `/learners/lessons`
- [x] 6.2 Add one assertion that `/learners/calendar` resolves to `/learners/lessons`
- [x] 6.3 Add one assertion that learner login opens `/learners` and shows the Start screen

## 7. Documentation and verification

- [x] 7.1 Add `docs/constitutions/learner-content.md` and link it from `docs/README.md` and `docs/api/materials.md`
- [x] 7.2 Add a learner navigation section to `docs/constitutions/frontend-view-states.md`
- [x] 7.3 Update learner route spaces in `docs/auth.md`
- [x] 7.4 Run `./tools/code_quality/check.py check`, `npm run check`, the Playwright specs, and `openspec validate learner-workspace-foundation --strict`
