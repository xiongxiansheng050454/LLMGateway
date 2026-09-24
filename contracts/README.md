# API Contract

`openapi.yaml` is the executable source for the Dashboard HTTP contract.

The frontend generates `dashboard-react/src/api/generated/schema.ts` from this
file. Generated output is committed and CI runs `npm run api:check` to ensure it
can be reproduced without drift.

The Go backend keeps ownership of business DTOs and rules. Its HTTP contract
tests are the runtime implementation check; the OpenAPI document owns the
external paths, methods, request shapes, response shapes, and enums.

## Local workflow

From `dashboard-react/`:

```text
npm run api:generate
npm run api:check
npm run typecheck
```

Update `openapi.yaml` first, regenerate the schema, then update the backend and
the domain API wrapper as needed. Do not edit generated files directly.
