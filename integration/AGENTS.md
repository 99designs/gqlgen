# INTEGRATION HARNESS KNOWLEDGE BASE

## OVERVIEW
`integration/` is a cross-language e2e harness: a Go gqlgen server (`integration/server`) tested by Node/Vitest GraphQL clients (`integration/src`).

## STRUCTURE
```text
integration/
├── server/                  # Go gqlgen integration server
│   ├── cmd/integration/     # Server entrypoint
│   ├── schema/              # GraphQL schemas
│   └── gqlgen.yml           # Server generation config
├── src/__test__/            # Vitest e2e tests
├── src/queries/             # GraphQL documents used by tests
├── src/generated/           # TS generated GraphQL types/doc nodes
└── package.json             # npm scripts + dependencies
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Boot integration server | `server/cmd/integration/server.go` | Binds HTTP endpoint used by tests |
| Server schema behavior | `server/schema/*.graphql` | Inputs for Go-side generation |
| Client e2e behavior | `src/__test__/integration.spec.ts` | HTTP + WS + SSE checks |
| Type/doc generation | `codegen.ts`, `src/generated/` | TS GraphQL type layer |

## CONVENTIONS
- Run server and test runner in separate terminals/processes.
- Keep Node generated artifacts aligned with query/schema changes.
- Treat this directory as integration-only: no unrelated runtime refactors.

## ANTI-PATTERNS (INTEGRATION)
- Running Vitest without a live Go server.
- Updating server schema without refreshing client generated outputs.
- Assuming Go-only checks cover this directory.

## COMMANDS
```bash
go run integration/server/cmd/integration/server.go
cd integration && npm ci && npm run test
```
