## Why

A learner cannot see in one sentence how much to pay, by which date, and to which account. The learner plan summary shows only counters, and the teacher shares bank details outside the application. Nutka targets adults and seniors, so the payment information must be plain and complete. The ledger already holds every value that the learner needs.

## What Changes

- Add a learner payment-due read for one assignment: open items, total, next contract forecast, recent payments, and the teacher transfer details.
- Add teacher transfer details (account holder, Polish IBAN, note) with read and update routes for the signed-in teacher.
- Add `/learners/payments` with the amount due, the due date, overdue badges, copy controls, a ZBP transfer QR code, the next month forecast, and recent payments.
- Add a Start screen card that shows the amount due only when it is above zero.
- Add `/teachers/settings` with the transfer details form.
- Keep the ledger non-enforcing. The screen never blocks booking or lesson changes.

## Capabilities

### New Capabilities

- `learner-payment-due`: The learner read of open payment items, total, forecast, recent payments, and transfer details, and its presentation.
- `teacher-payment-details`: Teacher transfer details, their validation, and their visibility.

### Modified Capabilities

None.

## Impact

- Backend: migration that adds hidden transfer fields to `teachers`, new `internal/paymentdetails` and `internal/paymentdetailsapi`, `ledger.DueItems`, and a new `ledgerapi` route.
- Frontend: `api/ledger.ts`, `query/keys.ts`, learner payments screen, Start card, teacher settings screen, and one new dependency for QR rendering (`qrcode-generator`).
- Docs: `docs/api/payments.md`, `docs/constitutions/frontend-view-states.md`.
