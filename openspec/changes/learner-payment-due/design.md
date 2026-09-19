## Context

See `proposal.md` for motivation and `specs/` for required behavior.

- The `charges` collection holds both contract month charges and ad hoc lesson charges. `source_type` tells them apart.
- `ledger.Charge.IsOverdue` is the one overdue rule. It uses the teacher timezone.
- Credits already reduce open charges when the ledger creates or adjusts a charge. The remaining open credit is leftover value.
- `ledgerapi.PocketBaseService.ContractMonths` already projects forecasts and charges per contract month.
- `ledgerapi` resolves its own caller. `internal/personaroute` is the shared rule for new routes.

## Goals / Non-Goals

**Goals:**

- Keep the open-item rule in the `ledger` package next to the overdue rule.
- Return machine values only. The client builds Polish copy, the transfer title, and the QR payload.

**Non-Goals:**

- Process, match, or confirm bank payments.
- Store per-learner transfer titles or payment references.
- Move the other `ledgerapi` routes to `personaroute`.

## Decisions

### Open-item rule in `ledger`

`ledger.DueItems(charges []DueCharge, now, location) ([]DueItem, int64)` holds the open-item rule, the order, and the total. `DueCharge` wraps `Charge` with `SourceType`, `LessonStartAt`, and `LessonEnded`. The rule is pure and has table tests.

Alternative: a new `paymentdue` package. Rejected because the rule reads only charge states, which the ledger owns.

### Route in `ledgerapi`

`GET /api/learners/assignments/{id}/payment-due` joins the ledger routes. `Service` gains `PaymentDue(ctx, actor, assignmentID)`. The PocketBase service reuses `assignmentForActor`, `chargesForAssignments`, `availableCredits`, and `ContractMonths`.

### Total does not subtract credit

The ledger applies credits to charges when it creates or adjusts them. Subtracting the open credit again would count it twice. The read shows the credit as a separate value.

### Transfer details on the `teachers` record

A migration adds hidden text fields `payment_account_holder`, `payment_iban`, and `payment_note` to `teachers`. Hidden fields stay out of auth responses and native routes. Each teacher has one account, so the details do not belong to the global business policy.

`internal/paymentdetails` normalizes and validates the values and has no PocketBase dependency. `internal/paymentdetailsapi` serves `GET` and `PUT /api/teachers/payment-details` through `personaroute`. `ledgerapi` reads the details of the assignment teacher through `paymentdetails.FromRecord`.

### Client-side title and QR payload

`apps/app/src/views/learner/payments/transfer.ts` builds the transfer title and the ZBP QR payload as pure functions. The title names the learner and the open months or lesson dates and holds at most 32 characters for the QR code. The payload follows the ZBP two-dimensional code recommendation: `|PL|<26 digits>|<amount in grosze, 6 digits>|<holder, max 20>|<title, max 32>|||`. The QR code appears only when the total is between 0.01 and 9999.99 PLN, which the six-digit amount field allows.

### QR rendering with `qrcode-generator`

`qrcode-generator` has no dependencies and returns an SVG string. The component renders it through an `img` element with a data URL, so the page inserts no raw HTML.

Alternative: `qrcode`. Rejected because it pulls command-line dependencies.

### Cache key under the assignment summary

The payment-due key sits under the assignment summary prefix. Every existing learner booking, lifecycle, and plan mutation already invalidates that prefix, so the payment-due read refreshes after them without new cache rules.

## Risks / Trade-offs

- [A bank app rejects the QR code] → The copy controls stay next to the QR code, and the IBAN and title stay visible as text.
- [The teacher records a payment late] → The screen says that the teacher records payments manually and that a recent transfer may still show as due.
- [The contract months read reconciles months on read] → The payment-due read calls it only for the assignment contract, the same as the existing months route.

## Migration Plan

- The migration adds three optional fields and needs no backfill.
- Rollback leaves the fields in place. No other record depends on them.
