# Non-Negotiables

## Layer Boundaries

1. `src/api/**` is the only place for PocketBase network operations:
   - Allowed only there: `pb.collection(...)`, `pb.send(...)`.
2. Forbidden outside `src/api/**`:
   - `pb.collection(...)`
   - `pb.send(...)`
   - raw `fetch(...)`
   - `axios(...)`
3. `pb.authStore` reads are allowed outside API services when required.

## API Layer Shape

1. Use `src/api/<entity>/routes.ts` with `default export class <Entity>Api`.
2. Use `src/api/<entity>/dtos.ts` for DTOs when needed.
3. Keep method names explicit (`list`, `getOne`, `add`, `update`, `delete`, domain-specific methods).
4. Keep query/filter construction close to API methods and typed.

## Hook Layer

1. Place server-state hooks in `src/modules/<module>/hooks/*`.
2. Hooks call API services only; no direct PocketBase network calls.
3. Hooks own:
   - query/mutation setup
   - cache invalidation/refetch coordination
4. Query keys must use central factories/constants.

## Component Layer

1. Components are presentation and interaction orchestration.
2. Keep component-side effects in components:
   - toasts
   - navigation
   - dialog/sheet open-close
3. Do not fetch in `useEffect`.
4. Avoid prop drilling where composition can simplify ownership.
5. Favor composition over monoliths: break large mixed-responsibility views into focused section/presentational components when touched.
6. Keep transformation logic in pure helpers or module hooks when JSX becomes dense.

## Types

1. Prefer `src/lib/pocketbase-types.ts` first.
2. Avoid creating interfaces already represented by generated types.
3. Use DTOs/custom types only for transformed/projection-specific shapes.
4. Never use `any`.

## Dashboard

Apply the same architecture rules as every other module:
- Hooks in module hook layer.
- PocketBase network access in API layer.
