# Refactor Playbook

Use this playbook when editing frontend code in `sailormoon.ui`.

## 1. Quick Discovery

Run targeted scans for touched scope:

```sh
rg -n "pb\\.collection\\(|pb\\.send\\(|fetch\\(|axios\\(" sailormoon.ui/src
rg -n "useQuery\\(|useMutation\\(" sailormoon.ui/src/modules sailormoon.ui/src/pages
```

## 2. Move Network Calls to API Services

If touched code performs network calls outside `src/api/**`:
1. Add/extend an API class method in `src/api/<entity>/routes.ts`.
2. Keep request-specific types in `dtos.ts` when necessary.
3. Replace direct network call sites with API method usage.

## 3. Move Server-State Logic to Module Hooks (Reasonably)

If touched components contain query/mutation logic:
1. Extract/extend hook in `src/modules/<module>/hooks/*`.
2. Hook should return data, loading/error state, and action functions.
3. Keep side-effects (toast/navigation/dialog) in the component.
4. Keep invalidation and cache strategy in the hook.

This is a strong refactor direction, not a mandatory full rewrite of untouched areas.

## 4. Improve Component Composability (Reasonably)

When touched components become large or mix multiple concerns:
1. Extract repeated or domain-specific UI blocks into focused components (for example, `*-section.tsx`, `*-chart.tsx`).
2. Move heavy data-to-view mapping into pure helpers or module hooks.
3. Keep parent components as orchestrators (state + composition), not giant render bodies.
4. Preserve behavior, i18n keys, and `data-testid` contracts.

## 5. Standardize Query Keys Centrally

Use a central query-key location (single shared place in the app), with entity factories:

```ts
export const boatsKeys = {
  all: () => ["boats"] as const,
  list: (params: BoatsListParams) => [...boatsKeys.all(), "list", params] as const,
  detail: (id: string) => [...boatsKeys.all(), "detail", id] as const,
};
```

Preferred naming shape:
- `<entity>Keys.all()`
- `<entity>Keys.list(params)`
- `<entity>Keys.detail(id)`

## 6. Type Discipline

1. Prefer generated types from `src/lib/pocketbase-types.ts`.
2. Add DTOs only when shape differs from generated records/responses.
3. Keep function signatures explicit and narrow.

## 7. Behavioral Safety

1. Preserve existing UI behavior unless task asks otherwise.
2. Preserve i18n keys and locale coverage for copy changes.
3. Preserve existing `data-testid` contracts.
4. Do not set `marina` in UI forms (backend-managed).

## 8. Verification

Run the most relevant checks for changed scope:
1. `pnpm -C sailormoon.ui lint`
2. `pnpm -C sailormoon.ui build`
3. Targeted tests for touched modules (unit/e2e as appropriate)
