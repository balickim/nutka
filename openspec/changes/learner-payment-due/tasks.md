## 1. Transfer details backend

- [x] 1.1 Add `internal/paymentdetails` with `Normalize`, `Validate`, and `FromRecord`, and tests for spaces, missing prefix, checksum, holder rule, and limits
- [x] 1.2 Add a migration with hidden `payment_account_holder`, `payment_iban`, and `payment_note` fields on `teachers`
- [x] 1.3 Add `internal/paymentdetailsapi` with `GET` and `PUT /api/teachers/payment-details` through `personaroute`, and register it in `main.go`
- [x] 1.4 Add an HTTP test: learner session gets `403`, invalid IBAN gets `400`, valid save round trips, and `/api/teachers/auth/me` hides the fields

## 2. Payment-due backend

- [x] 2.1 Add `ledger.DueItems` with table tests for open states, future ad hoc lessons, overdue order, and total
- [x] 2.2 Add `PaymentDue` to `ledgerapi.Service`, the PocketBase implementation, the DTOs, and the learner route
- [x] 2.3 Add an HTTP test: foreign learner gets `403`, open contract charge and ended ad hoc charge appear, paid charge appears in recent payments, and instructions follow the teacher details
- [x] 2.4 Run `go build ./...` and `go test ./...`

## 3. Frontend data

- [x] 3.1 Add payment-due and payment-details types and requests to `api/ledger.ts`
- [x] 3.2 Add query keys, queries, and the teacher details mutation with its cache effect, and extend `keys.test.ts`
- [x] 3.3 Add pure `transferTitle` and `zbpPayload` with tests for length limits, amount padding, and IBAN digits

## 4. Learner screens

- [x] 4.1 Add `qrcode-generator` and run `npm audit --omit=dev --audit-level=high`
- [x] 4.2 Add `/learners/payments` with total, items, overdue badges, transfer details, copy controls, QR code, forecast, and recent payments
- [x] 4.3 Add the Start payment card shown only for a total above zero
- [x] 4.4 Add a payments link to the lessons plan card

## 5. Teacher settings

- [x] 5.1 Add `/teachers/settings` with the transfer details form and the navigation entry

## 6. Tests, docs, and verification

- [x] 6.1 Add a Playwright flow: learner with a pending charge sees the Start card, opens payments, and sees the IBAN, title, and QR code
- [x] 6.2 Update `docs/api/payments.md` and `docs/constitutions/frontend-view-states.md`
- [x] 6.3 Run `./tools/code_quality/check.py check`, `npm run check`, vitest, Playwright, and `openspec validate learner-payment-due --strict`
