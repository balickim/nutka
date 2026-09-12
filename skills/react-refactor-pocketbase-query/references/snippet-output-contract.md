# Snippet Output Contract

Apply this contract only when generating example code/snippets in chat.

## Required Order

1. Assumptions
2. Query keys to be used
3. Types/Interfaces
4. API Service (`src/api/<entity>/routes.ts` and `dtos.ts` style)
5. Custom Hook (`src/modules/<module>/hooks/*`)
6. Component

## Additional Rules

1. Provide complete logic; no placeholder comments.
2. Keep compatibility with class-based API service pattern.
3. Do not force this sectioned format when editing real repository files in place.
4. For complex UI snippets, prefer parent-orchestrator + composed child sections instead of one large component.
