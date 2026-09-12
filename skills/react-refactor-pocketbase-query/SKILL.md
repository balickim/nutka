---
name: react-refactor-pocketbase-query
description: Refactor React/TypeScript code toward loose coupling and composability with class-based API services, module hooks, standardized TanStack Query keys, and PocketBase network access only in src/api.
---

# React Refactor PocketBase Query

Use this skill when working on React/TypeScript code that should be maintainable, loosely coupled, and consistent with the repository architecture.

## Load Minimum Context First

1. Read `AGENTS.md`.
2. Read `docs/README.md`.
3. Read only constitutions relevant to touched modules under `docs/constitutions/**`.
4. Load detailed rules only when needed:
   - [references/non-negotiables.md](references/non-negotiables.md)
   - [references/refactor-playbook.md](references/refactor-playbook.md)
   - [references/snippet-output-contract.md](references/snippet-output-contract.md)

## Architecture Baseline (Do Not Redesign)

- Keep `src/api/<entity>/routes.ts` + `src/api/<entity>/dtos.ts`.
- Keep class-based API clients: `default export class <Entity>Api` with explicit methods.
- Keep module hook placement in `src/modules/<module>/hooks/*`.
- Prefer composable UI sections/components over large monolithic views when touched.

## Required Behavior

1. All PocketBase network calls (`pb.collection`, `pb.send`) must live in `src/api/**` only.
2. Components/hooks/libs/pages must not call PocketBase network APIs directly.
3. Components should consume module hooks; move direct `useQuery`/`useMutation` from components into hooks when touched and reasonable.
4. Improve composability in touched views: split large multi-responsibility components into focused section components and/or mapping hooks when reasonable.
5. Standardize query keys through central factories/constants, using `<entity>Keys.list(params)` style.
6. Keep mutation side-effects in components (toast/navigation/dialog), while hooks own data/cache logic.
7. Prefer generated types/enums from `src/lib/pocketbase-types.ts`; do not duplicate them.
8. Treat dashboard like every other module: hooks in module layer, data access in API layer.

## Execution Workflow

1. Scan touched files for violations against non-negotiables.
2. If a non-API file performs network calls, move those calls into the relevant API class.
3. If a component contains heavy query/mutation logic, extract/refine a module hook (reasonable scope only).
4. If a component is large or mixes data mapping with dense JSX, extract composable section components and/or pure mapping utilities without changing behavior.
5. Introduce or update central query-key factories and migrate touched call sites.
6. Keep UI behavior unchanged unless task explicitly requests behavior changes.
7. Run relevant checks (`lint`, targeted tests, build as needed) before finishing.

## Communication Requirements

- When requirements are ambiguous and impact architecture, ask focused clarifying questions using a numbered list.
- Explain assumptions explicitly when making them.
- If writing sample code in chat, apply the output contract in `references/snippet-output-contract.md`.
