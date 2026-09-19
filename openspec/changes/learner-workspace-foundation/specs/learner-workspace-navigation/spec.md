## Purpose

Defines the learner route spaces, what each learner screen answers, the data each screen loads, the assignment switcher, and readable learner copy for dates and payments.

## ADDED Requirements

### Requirement: Learner routes share one guarded shell

Every learner screen SHALL render inside one shell that requires the learner session. The shell SHALL load the learner session, business policy, and learner calendar once. The shell SHALL render navigation to every available learner screen and a logout control.

#### Scenario: Guest opens a learner screen

- **WHEN** a guest opens `/learners`, `/learners/lessons`, or `/learners/pieces`
- **THEN** the application redirects to `/learners/login` with the requested location as the redirect target

#### Scenario: Learner moves between screens

- **WHEN** an authenticated learner moves from `/learners` to `/learners/lessons`
- **THEN** the shell keeps the loaded session, policy, and calendar and does not request them again

### Requirement: Learner home is the Start screen

After learner login, the application SHALL open `/learners`. The Start screen SHALL display the next scheduled lesson of the selected assignment and links to the lessons and pieces screens. The Start screen SHALL NOT render booking, notice, or history controls.

#### Scenario: Learner logs in

- **WHEN** a learner logs in without a redirect target
- **THEN** the application opens `/learners`

#### Scenario: Learner has no upcoming lesson

- **WHEN** the selected assignment has no scheduled future lesson
- **THEN** the Start screen shows one sentence and one control that opens `/learners/lessons`

### Requirement: Lessons screen holds lesson and plan work

`/learners/lessons` SHALL display the lesson list, flexible booking, plan summary, contract notice, history, and package token details for the selected assignment. It SHALL keep every existing booking, rescheduling, cancellation, and notice rule.

#### Scenario: Learner with a regular contract opens lessons

- **WHEN** the selected assignment has an active regular contract
- **THEN** the lessons screen shows the contract summary and the notice control and hides flexible booking

### Requirement: Pieces screen holds teacher materials

`/learners/pieces` SHALL display the teacher materials of the selected assignment. It SHALL load materials only on this screen.

#### Scenario: Learner opens Start

- **WHEN** an authenticated learner opens `/learners`
- **THEN** the application does not request the materials list

### Requirement: Assignment switcher appears for multiple teachers

The shell SHALL select the assignment from the `a` search parameter. It SHALL fall back to the first active assignment when the parameter is missing or does not match an active assignment. The shell SHALL render an assignment switcher only when the learner has more than one active assignment.

#### Scenario: Learner has one teacher

- **WHEN** the learner has exactly one active assignment
- **THEN** the shell renders no assignment switcher and every screen uses that assignment

#### Scenario: Learner switches teacher

- **WHEN** the learner with two active assignments selects the second teacher
- **THEN** the location gains `a=<assignment id>` and every screen shows data for that assignment

#### Scenario: Learner has no active assignment

- **WHEN** the learner has no active assignment
- **THEN** every screen shows one sentence that tells the learner to contact the teacher

### Requirement: Learner copy uses readable dates and payment facts

The learner screens SHALL display date-only plan values as long Polish dates, for example `wtorek, 14 października 2026`. The plan summary SHALL display a payment fact only when its value is not zero.

#### Scenario: Package validity date

- **WHEN** a package has `valid_through` equal to `2026-10-14`
- **THEN** the lessons screen shows `wtorek, 14 października 2026` and never shows `2026-10-14`

#### Scenario: No open payments

- **WHEN** pending, unpaid, and credit values are all zero
- **THEN** the plan summary shows no payment line
